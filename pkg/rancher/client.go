// Package rancher provides a client for resolving Rancher cluster targets and
// building per-cluster REST configs that proxy through Rancher's cattle-cluster-agent tunnel.
package rancher

import (
	"context"
	"crypto/x509"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	secretsv1alpha1 "github.com/txodds/rancher-secrets-manager/pkg/api/v1alpha1"
)

var clusterGVR = schema.GroupVersionResource{
	Group:    "management.cattle.io",
	Version:  "v3",
	Resource: "clusters",
}

// ResolvedTarget is a concrete (clusterID, clusterName, namespace, secretName) tuple
// derived from expanding a ManagedSecret Target entry.
type ResolvedTarget struct {
	ClusterID   string
	ClusterName string
	Namespace   string
	SecretName  string
}

// Config holds the configuration for the Rancher client.
type Config struct {
	// RancherURL is the base URL of the Rancher API, e.g.
	// https://rancher.cattle-system.svc or https://rancher.example.com.
	RancherURL string
	// InsecureTLS skips TLS certificate verification. Use only in development.
	InsecureTLS bool
	// CABundlePath is an optional path to a PEM CA bundle to trust for the Rancher endpoint.
	CABundlePath string
}

// Client wraps the Rancher management API and builds downstream cluster configs.
type Client struct {
	cfg           Config
	dynamicClient dynamic.Interface
	// token is the controller's service account token used to authenticate to Rancher.
	token string
}

// NewClient creates a Client using the given Rancher config and the in-cluster REST config
// (used to build the dynamic client for listing management.cattle.io clusters).
func NewClient(cfg Config, restCfg *rest.Config) (*Client, error) {
	dyn, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("building dynamic client: %w", err)
	}

	token, err := saToken()
	if err != nil {
		return nil, fmt.Errorf("reading service account token: %w", err)
	}

	if cfg.CABundlePath != "" {
		if err := validateCABundle(cfg.CABundlePath); err != nil {
			return nil, fmt.Errorf("CA bundle: %w", err)
		}
	}

	return &Client{
		cfg:           cfg,
		dynamicClient: dyn,
		token:         token,
	}, nil
}

// ResolveTargets expands the target list from a ManagedSecret into concrete
// (clusterID, namespace, secretName) triples.
func (c *Client) ResolveTargets(ctx context.Context, srcSecretName string, targets []secretsv1alpha1.Target) ([]ResolvedTarget, error) {
	allClusters, err := c.listClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing Rancher clusters: %w", err)
	}

	var resolved []ResolvedTarget
	seen := map[string]bool{} // deduplicate (clusterID, namespace) pairs

	for _, target := range targets {
		secretName := target.SecretName
		if secretName == "" {
			secretName = srcSecretName
		}

		matched, err := c.matchClusters(allClusters, target)
		if err != nil {
			return nil, fmt.Errorf("matching clusters for target: %w", err)
		}

		for _, cl := range matched {
			key := cl.ClusterID + "/" + target.Namespace
			if seen[key] {
				continue
			}
			seen[key] = true
			resolved = append(resolved, ResolvedTarget{
				ClusterID:   cl.ClusterID,
				ClusterName: cl.ClusterName,
				Namespace:   target.Namespace,
				SecretName:  secretName,
			})
		}
	}

	return resolved, nil
}

// BuildClusterConfig returns a *rest.Config that routes requests through Rancher's
// API proxy for the given cluster ID, authenticated with the controller's SA token.
func (c *Client) BuildClusterConfig(clusterID string) (*rest.Config, error) {
	host := fmt.Sprintf("%s/k8s/clusters/%s", c.cfg.RancherURL, clusterID)
	cfg := &rest.Config{
		Host:        host,
		BearerToken: c.token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: c.cfg.InsecureTLS,
			CAFile:   c.cfg.CABundlePath,
		},
	}
	return cfg, nil
}

// clusterInfo is a minimal representation of a Rancher cluster.
type clusterInfo struct {
	ClusterID   string
	ClusterName string
	Labels      map[string]string
}

func (c *Client) listClusters(ctx context.Context) ([]clusterInfo, error) {
	list, err := c.dynamicClient.Resource(clusterGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusters := make([]clusterInfo, 0, len(list.Items))
	for _, item := range list.Items {
		id := item.GetName()
		displayName := displayNameOf(item)
		clusters = append(clusters, clusterInfo{
			ClusterID:   id,
			ClusterName: displayName,
			Labels:      item.GetLabels(),
		})
	}
	return clusters, nil
}

func (c *Client) matchClusters(all []clusterInfo, target secretsv1alpha1.Target) ([]clusterInfo, error) {
	if target.ClusterName != "" {
		for _, cl := range all {
			if cl.ClusterID == target.ClusterName || cl.ClusterName == target.ClusterName {
				return []clusterInfo{cl}, nil
			}
		}
		return nil, fmt.Errorf("no cluster found with name or ID %q", target.ClusterName)
	}

	if target.ClusterSelector == nil {
		return nil, fmt.Errorf("target must specify clusterName or clusterSelector")
	}

	sel, err := metav1.LabelSelectorAsSelector(target.ClusterSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid clusterSelector: %w", err)
	}

	var matched []clusterInfo
	for _, cl := range all {
		if sel.Matches(labelsAsSet(cl.Labels)) {
			matched = append(matched, cl)
		}
	}
	return matched, nil
}

// displayNameOf extracts the human-readable cluster name from the unstructured object.
// Rancher stores it in spec.displayName for management.cattle.io/v3 Cluster resources.
func displayNameOf(obj unstructured.Unstructured) string {
	if name, ok, _ := unstructured.NestedString(obj.Object, "spec", "displayName"); ok && name != "" {
		return name
	}
	return obj.GetName()
}

func saToken() (string, error) {
	const tokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", tokenPath, err)
	}
	return string(data), nil
}

func validateCABundle(path string) error {
	pem, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %q: %w", path, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return fmt.Errorf("no valid certificates found in %q", path)
	}
	return nil
}

// labelsAsSet wraps a map so it satisfies labels.Labels for selector matching.
type labelsAsSet map[string]string

func (l labelsAsSet) Has(key string) bool   { _, ok := l[key]; return ok }
func (l labelsAsSet) Get(key string) string { return l[key] }
