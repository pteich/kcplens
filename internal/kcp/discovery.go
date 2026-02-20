package kcp

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type AvailableResource struct {
	GVR        schema.GroupVersionResource
	Kind       string
	Namespaced bool
	Count      int
}

type APIRelationship struct {
	Name          string
	Type          string // "Export" or "Binding"
	Status        string
	ExportName    string // For bindings: the export name it binds to
	ExportPath    string // For bindings: the workspace path of the export
	ResourceName  string // For exports: the resource being exported
	ResourceGroup string
	Raw           map[string]interface{} // Raw object for YAML display
}

type SyncTarget struct {
	Name   string
	Status string
	Labels map[string]string
}

type GenericResource struct {
	Name      string
	Namespace string
	Kind      string
	Workspace string
}

type WorkspaceNode struct {
	Name     string
	Path     string
	Children []*WorkspaceNode
}

// DiscoverWorkspaces lists workspaces under a given path, using cache if available.
func (c *ClientManager) DiscoverWorkspaces(ctx context.Context, parentPath string) ([]*WorkspaceNode, error) {
	if parentPath == "" {
		parentPath = "root"
	}

	// Check cache
	if res, ok := c.discoveryCache[parentPath]; ok {
		if nodes, ok := res.([]*WorkspaceNode); ok {
			return nodes, nil
		}
	}

	// Switch to the parent workspace to list children
	if err := c.SwitchWorkspace(parentPath); err != nil {
		return nil, fmt.Errorf("failed to switch to workspace %s: %w", parentPath, err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "tenancy.kcp.io",
		Version:  "v1alpha1",
		Resource: "workspaces",
	}

	workspaceList, err := c.DynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces in %s: %w", parentPath, err)
	}

	var nodes []*WorkspaceNode
	for _, ws := range workspaceList.Items {
		nodes = append(nodes, &WorkspaceNode{
			Name: ws.GetName(),
			Path: parentPath + ":" + ws.GetName(),
		})
	}

	// Update cache
	c.discoveryCache[parentPath] = nodes

	return nodes, nil
}

// DiscoverRootWorkspaces is a convenience wrapper for root discovery.
func (c *ClientManager) DiscoverRootWorkspaces(ctx context.Context) ([]*WorkspaceNode, error) {
	return c.DiscoverWorkspaces(ctx, "root")
}

// IsRoot returns true if the current path is the root cluster.
func IsRoot(path string) bool {
	return path == "root"
}

// ParentPath returns the parent of a given workspace path.
func ParentPath(path string) string {
	if path == "root" || !strings.Contains(path, ":") {
		return "root"
	}
	idx := strings.LastIndex(path, ":")
	return path[:idx]
}

// DiscoverAPIRelationships lists APIExports and APIBindings in the current workspace.
func (c *ClientManager) DiscoverAPIRelationships(ctx context.Context, path string) ([]APIRelationship, error) {
	if err := c.SwitchWorkspace(path); err != nil {
		return nil, err
	}

	var relationships []APIRelationship

	for _, version := range []string{"v1alpha2", "v1alpha1"} {
		exportGVR := schema.GroupVersionResource{
			Group:    "apis.kcp.io",
			Version:  version,
			Resource: "apiexports",
		}
		exports, err := c.DynamicClient.Resource(exportGVR).List(ctx, metav1.ListOptions{})
		if err == nil && len(exports.Items) > 0 {
			for _, item := range exports.Items {
				rel := APIRelationship{
					Name:   item.GetName(),
					Type:   "Export",
					Status: getStatus(item),
					Raw:    item.Object,
				}
				if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
					if resources, ok := spec["resources"].([]interface{}); ok && len(resources) > 0 {
						if r, ok := resources[0].(map[string]interface{}); ok {
							rel.ResourceName, _ = r["name"].(string)
							rel.ResourceGroup, _ = r["group"].(string)
						}
					}
				}
				relationships = append(relationships, rel)
			}
			break
		}
	}

	for _, version := range []string{"v1alpha2", "v1alpha1"} {
		bindingGVR := schema.GroupVersionResource{
			Group:    "apis.kcp.io",
			Version:  version,
			Resource: "apibindings",
		}
		bindings, err := c.DynamicClient.Resource(bindingGVR).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}
		if len(bindings.Items) > 0 {
			for _, item := range bindings.Items {
				rel := APIRelationship{
					Name:   item.GetName(),
					Type:   "Binding",
					Status: getStatus(item),
					Raw:    item.Object,
				}
				if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
					if ref, ok := spec["reference"].(map[string]interface{}); ok {
						if exp, ok := ref["export"].(map[string]interface{}); ok {
							rel.ExportName, _ = exp["name"].(string)
							rel.ExportPath, _ = exp["path"].(string)
						}
					}
				}
				relationships = append(relationships, rel)
			}
			break
		}
	}

	return relationships, nil
}

func getStatus(u unstructured.Unstructured) string {
	status, found, _ := unstructured.NestedMap(u.Object, "status")
	if !found {
		return "Unknown"
	}

	if phase, ok := status["phase"].(string); ok && phase != "" {
		return phase
	}

	conditions, found, _ := unstructured.NestedSlice(status, "conditions")
	if !found || len(conditions) == 0 {
		return "Unknown"
	}

	for _, cond := range conditions {
		if c, ok := cond.(map[string]interface{}); ok {
			if t, ok := c["type"].(string); ok && t == "Ready" {
				if s, ok := c["status"].(string); ok {
					if s == "True" {
						return "Ready"
					}
					return "NotReady"
				}
			}
		}
	}

	if len(conditions) > 0 {
		if lastCond, ok := conditions[len(conditions)-1].(map[string]interface{}); ok {
			if t, ok := lastCond["type"].(string); ok {
				return t
			}
		}
	}

	return "Unknown"
}

// DiscoverSyncTargets lists SyncTargets in the current workspace.
func (c *ClientManager) DiscoverSyncTargets(ctx context.Context, path string) ([]SyncTarget, error) {
	if err := c.SwitchWorkspace(path); err != nil {
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    "workload.kcp.io",
		Version:  "v1alpha1",
		Resource: "synctargets",
	}

	list, err := c.DynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var targets []SyncTarget
	for _, item := range list.Items {
		targets = append(targets, SyncTarget{
			Name:   item.GetName(),
			Status: getStatus(item),
			Labels: item.GetLabels(),
		})
	}

	return targets, nil
}

// DiscoverResources lists resources of a specific GVR in a workspace.
func (c *ClientManager) DiscoverResources(ctx context.Context, path string, gvr schema.GroupVersionResource) ([]GenericResource, error) {
	if err := c.SwitchWorkspace(path); err != nil {
		return nil, err
	}

	list, err := c.DynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []GenericResource
	for _, item := range list.Items {
		resources = append(resources, GenericResource{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
			Kind:      item.GetKind(),
			Workspace: path,
		})
	}

	return resources, nil
}

// DiscoverWildcardResources lists resources across all workspaces using clusters/*.
func (c *ClientManager) DiscoverWildcardResources(ctx context.Context, gvr schema.GroupVersionResource) ([]GenericResource, error) {
	savedWorkspace := c.currentWorkspace

	wildcardHost := c.baseHost + "/clusters/*"
	cfg := rest.CopyConfig(c.RestConfig)
	cfg.Host = wildcardHost

	wildcardClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	list, err := wildcardClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.SwitchWorkspace(savedWorkspace)
		return nil, err
	}

	var resources []GenericResource
	for _, item := range list.Items {
		ws := "unknown"
		if cluster := item.GetAnnotations()["kcp.io/cluster"]; cluster != "" {
			ws = cluster
		}

		resources = append(resources, GenericResource{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
			Kind:      item.GetKind(),
			Workspace: ws,
		})
	}

	c.SwitchWorkspace(savedWorkspace)

	return resources, nil
}

// DiscoverAvailableResources finds all available API resources in a workspace.
func (c *ClientManager) DiscoverAvailableResources(ctx context.Context, path string) ([]AvailableResource, error) {
	if err := c.SwitchWorkspace(path); err != nil {
		return nil, err
	}

	apiResourceLists, err := c.DiscoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("failed to discover resources: %w", err)
	}

	var available []AvailableResource
	seen := make(map[string]bool)

	for _, apiList := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiList.GroupVersion)
		if err != nil {
			continue
		}

		for _, r := range apiList.APIResources {
			if strings.Contains(r.Name, "/") {
				continue
			}

			if seen[r.Name+"."+gv.Group] {
				continue
			}
			seen[r.Name+"."+gv.Group] = true

			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: r.Name,
			}

			available = append(available, AvailableResource{
				GVR:        gvr,
				Kind:       r.Kind,
				Namespaced: r.Namespaced,
				Count:      -1,
			})
		}
	}

	return available, nil
}

// DiscoverResourcesInWorkspace lists resources of a specific type in a workspace.
func (c *ClientManager) DiscoverResourcesInWorkspace(ctx context.Context, path string, gvr schema.GroupVersionResource, namespace string) ([]GenericResource, error) {
	if err := c.SwitchWorkspace(path); err != nil {
		return nil, err
	}

	opts := metav1.ListOptions{}
	var list *unstructured.UnstructuredList
	var err error

	if namespace != "" {
		list, err = c.DynamicClient.Resource(gvr).Namespace(namespace).List(ctx, opts)
	} else {
		list, err = c.DynamicClient.Resource(gvr).List(ctx, opts)
	}
	if err != nil {
		return nil, err
	}

	var resources []GenericResource
	for _, item := range list.Items {
		resources = append(resources, GenericResource{
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
			Kind:      item.GetKind(),
			Workspace: path,
		})
	}

	return resources, nil
}
