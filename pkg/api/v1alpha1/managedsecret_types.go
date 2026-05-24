package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Source Secret",type=string,JSONPath=`.spec.secretRef.name`
// +kubebuilder:printcolumn:name="Targets",type=integer,JSONPath=`.status.targetCount`
// +kubebuilder:printcolumn:name="Synced",type=integer,JSONPath=`.status.syncedCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

type ManagedSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedSecretSpec   `json:"spec,omitempty"`
	Status ManagedSecretStatus `json:"status,omitempty"`
}

type ManagedSecretSpec struct {
	// SecretRef references the source Secret in the management cluster.
	SecretRef SecretReference `json:"secretRef"`

	// Targets defines which downstream clusters and namespaces receive the secret.
	// +kubebuilder:validation:MinItems=1
	Targets []Target `json:"targets"`
}

type SecretReference struct {
	// Name of the source Secret.
	Name string `json:"name"`
	// Namespace of the source Secret.
	Namespace string `json:"namespace"`
}

type Target struct {
	// ClusterName targets a specific downstream cluster by its Rancher display name or ID.
	// Mutually exclusive with ClusterSelector.
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// ClusterSelector targets all downstream clusters whose labels match.
	// Mutually exclusive with ClusterName.
	// +optional
	ClusterSelector *metav1.LabelSelector `json:"clusterSelector,omitempty"`

	// Namespace is the namespace in the downstream cluster where the secret will be created.
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// SecretName overrides the secret name in the downstream cluster.
	// Defaults to the source secret's name.
	// +optional
	SecretName string `json:"secretName,omitempty"`
}

type ManagedSecretStatus struct {
	// SyncStatus records the sync result for each resolved (cluster, namespace) pair.
	// +optional
	SyncStatus []ClusterSyncStatus `json:"syncStatus,omitempty"`

	// Conditions summarises the overall sync state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// TargetCount is the total number of resolved (cluster, namespace) targets.
	// +optional
	TargetCount int `json:"targetCount,omitempty"`

	// SyncedCount is the number of targets that are currently in Synced state.
	// +optional
	SyncedCount int `json:"syncedCount,omitempty"`
}

// ClusterSyncStatus records the last sync result for one (cluster, namespace) pair.
type ClusterSyncStatus struct {
	// ClusterName is the Rancher display name of the cluster.
	ClusterName string `json:"clusterName"`
	// ClusterID is the Rancher internal cluster ID (e.g. c-m-xxxx).
	ClusterID string `json:"clusterId"`
	// Namespace is the namespace in the downstream cluster.
	Namespace string `json:"namespace"`
	// SecretName is the name of the secret in the downstream cluster.
	SecretName string `json:"secretName"`
	// Status is the current sync state.
	Status SyncState `json:"status"`
	// Message contains a human-readable reason when Status is Failed.
	// +optional
	Message string `json:"message,omitempty"`
	// LastSyncTime is the timestamp of the most recent successful sync.
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// SyncState describes the sync result for a single target.
// +kubebuilder:validation:Enum=Synced;Failed;Pending
type SyncState string

const (
	SyncStateSynced  SyncState = "Synced"
	SyncStateFailed  SyncState = "Failed"
	SyncStatePending SyncState = "Pending"
)

// +kubebuilder:object:root=true

type ManagedSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedSecret{}, &ManagedSecretList{})
}
