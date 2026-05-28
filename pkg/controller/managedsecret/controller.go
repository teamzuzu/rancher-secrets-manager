package managedsecret

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	secretsv1alpha1 "github.com/txodds/rancher-secrets-manager/pkg/api/v1alpha1"
	"github.com/txodds/rancher-secrets-manager/pkg/rancher"
)

const (
	requeueAfter     = 30 * time.Second
	labelManagedBy   = "secrets.cattle.io/managed-by"
	annotationSource = "secrets.cattle.io/source-managed-secret"
)

// Reconciler reconciles ManagedSecret resources.
type Reconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	RancherClient *rancher.Client
}

// +kubebuilder:rbac:groups=secrets.cattle.io,resources=managedsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=secrets.cattle.io,resources=managedsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=management.cattle.io,resources=clusters,verbs=get;list;watch

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ms secretsv1alpha1.ManagedSecret
	if err := r.Get(ctx, req.NamespacedName, &ms); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch the source secret from the management cluster.
	var srcSecret corev1.Secret
	srcKey := types.NamespacedName{Name: ms.Spec.SecretRef.Name, Namespace: ms.Spec.SecretRef.Namespace}
	if err := r.Get(ctx, srcKey, &srcSecret); err != nil {
		if k8serrors.IsNotFound(err) {
			return r.setFailedCondition(ctx, &ms, fmt.Sprintf("source secret %s/%s not found", ms.Spec.SecretRef.Namespace, ms.Spec.SecretRef.Name))
		}
		return ctrl.Result{}, err
	}

	// Resolve all (cluster, namespace) targets.
	targets, err := r.RancherClient.ResolveTargets(ctx, srcSecret.Name, ms.Spec.Targets)
	if err != nil {
		return r.setFailedCondition(ctx, &ms, fmt.Sprintf("resolving targets: %v", err))
	}

	// Sync to each target.
	syncStatuses := make([]secretsv1alpha1.ClusterSyncStatus, 0, len(targets))
	syncedCount := 0
	for _, target := range targets {
		status := r.syncToCluster(ctx, target, &srcSecret, ms.Name)
		syncStatuses = append(syncStatuses, status)
		if status.Status == secretsv1alpha1.SyncStateSynced {
			syncedCount++
		}
	}

	// Update status.
	ms.Status.SyncStatus = syncStatuses
	ms.Status.TargetCount = len(targets)
	ms.Status.SyncedCount = syncedCount
	ms.Status.Conditions = buildConditions(syncStatuses, ms.Generation)

	if err := r.Status().Update(ctx, &ms); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating status: %w", err)
	}

	logger.Info("reconciled", "targets", len(targets), "synced", syncedCount)
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

func (r *Reconciler) syncToCluster(
	ctx context.Context,
	target rancher.ResolvedTarget,
	srcSecret *corev1.Secret,
	managedSecretName string,
) secretsv1alpha1.ClusterSyncStatus {
	logger := log.FromContext(ctx).WithValues("cluster", target.ClusterID, "namespace", target.Namespace)

	status := secretsv1alpha1.ClusterSyncStatus{
		ClusterName: target.ClusterName,
		ClusterID:   target.ClusterID,
		Namespace:   target.Namespace,
		SecretName:  target.SecretName,
		Status:      secretsv1alpha1.SyncStatePending,
	}

	clusterCfg, err := r.RancherClient.BuildClusterConfig(ctx, target.ClusterID)
	if err != nil {
		status.Status = secretsv1alpha1.SyncStateFailed
		status.Message = fmt.Sprintf("building cluster config: %v", err)
		return status
	}

	cs, err := kubernetes.NewForConfig(clusterCfg)
	if err != nil {
		status.Status = secretsv1alpha1.SyncStateFailed
		status.Message = fmt.Sprintf("creating downstream client: %v", err)
		return status
	}

	// Ensure the target namespace exists.
	if err := ensureNamespace(ctx, cs, target.Namespace); err != nil {
		status.Status = secretsv1alpha1.SyncStateFailed
		status.Message = fmt.Sprintf("ensuring namespace: %v", err)
		return status
	}

	desired := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      target.SecretName,
			Namespace: target.Namespace,
			Labels: map[string]string{
				labelManagedBy: managedSecretName,
			},
			Annotations: map[string]string{
				annotationSource: managedSecretName,
			},
		},
		Type: srcSecret.Type,
		Data: srcSecret.Data,
	}

	existing, err := cs.CoreV1().Secrets(target.Namespace).Get(ctx, target.SecretName, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			status.Status = secretsv1alpha1.SyncStateFailed
			status.Message = fmt.Sprintf("getting existing secret: %v", err)
			return status
		}
		// Create.
		if _, err := cs.CoreV1().Secrets(target.Namespace).Create(ctx, desired, metav1.CreateOptions{}); err != nil {
			status.Status = secretsv1alpha1.SyncStateFailed
			status.Message = fmt.Sprintf("creating secret: %v", err)
			return status
		}
		logger.Info("created secret")
	} else {
		// Update only if data or type changed.
		if secretNeedsUpdate(existing, desired) {
			existing.Data = desired.Data
			existing.Type = desired.Type
			existing.Labels = mergeLabels(existing.Labels, desired.Labels)
			existing.Annotations = mergeLabels(existing.Annotations, desired.Annotations)
			if _, err := cs.CoreV1().Secrets(target.Namespace).Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
				status.Status = secretsv1alpha1.SyncStateFailed
				status.Message = fmt.Sprintf("updating secret: %v", err)
				return status
			}
			logger.Info("updated secret")
		}
	}

	now := metav1.Now()
	status.Status = secretsv1alpha1.SyncStateSynced
	status.LastSyncTime = &now
	return status
}

// SetupWithManager registers the controller and configures watches.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Map a Secret change to all ManagedSecrets that reference it.
	secretMapper := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		var msList secretsv1alpha1.ManagedSecretList
		if err := r.List(ctx, &msList); err != nil {
			return nil
		}
		var requests []reconcile.Request
		for _, ms := range msList.Items {
			if ms.Spec.SecretRef.Name == obj.GetName() && ms.Spec.SecretRef.Namespace == obj.GetNamespace() {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{Name: ms.Name},
				})
			}
		}
		return requests
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1alpha1.ManagedSecret{}).
		Watches(
			&corev1.Secret{},
			secretMapper,
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

// ensureNamespace creates the namespace in the downstream cluster if it doesn't exist.
func ensureNamespace(ctx context.Context, cs kubernetes.Interface, ns string) error {
	_, err := cs.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !k8serrors.IsNotFound(err) {
		return err
	}
	_, err = cs.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func secretNeedsUpdate(existing, desired *corev1.Secret) bool {
	if existing.Type != desired.Type {
		return true
	}
	if len(existing.Data) != len(desired.Data) {
		return true
	}
	for k, v := range desired.Data {
		if string(existing.Data[k]) != string(v) {
			return true
		}
	}
	return false
}

func mergeLabels(base, overlay map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

func buildConditions(statuses []secretsv1alpha1.ClusterSyncStatus, generation int64) []metav1.Condition {
	allSynced := true
	for _, s := range statuses {
		if s.Status != secretsv1alpha1.SyncStateSynced {
			allSynced = false
			break
		}
	}

	readyStatus := metav1.ConditionTrue
	readyReason := "AllSynced"
	readyMessage := fmt.Sprintf("All %d targets synced", len(statuses))
	if !allSynced {
		readyStatus = metav1.ConditionFalse
		readyReason = "SyncFailed"
		readyMessage = "One or more targets failed to sync"
	}

	return []metav1.Condition{{
		Type:               "Ready",
		Status:             readyStatus,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             readyReason,
		Message:            readyMessage,
	}}
}

func (r *Reconciler) setFailedCondition(ctx context.Context, ms *secretsv1alpha1.ManagedSecret, msg string) (ctrl.Result, error) {
	ms.Status.Conditions = []metav1.Condition{{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: ms.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             "Error",
		Message:            msg,
	}}
	_ = r.Status().Update(ctx, ms)
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
