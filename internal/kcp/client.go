package kcp

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type ClientManager struct {
	RestConfig      *rest.Config
	Clientset       *kubernetes.Clientset
	DynamicClient   dynamic.Interface
	DiscoveryClient discovery.DiscoveryInterface
	baseHost        string

	currentWorkspace string
	discoveryCache   map[string]interface{}
}

func NewClientManager(kubeconfigPath string) (*ClientManager, error) {
	config, err := loadKubeConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	baseHost := config.Host
	if idx := strings.Index(config.Host, "/clusters/"); idx > 0 {
		baseHost = config.Host[:idx]
	}

	config.Host = baseHost + "/clusters/root"

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	return &ClientManager{
		RestConfig:       config,
		Clientset:        clientset,
		DynamicClient:    dynamicClient,
		DiscoveryClient:  discoveryClient,
		baseHost:         baseHost,
		currentWorkspace: "root",
		discoveryCache:   make(map[string]interface{}),
	}, nil
}

func NewClientManagerWithContext(kubeconfigPath, contextName string) (*ClientManager, error) {
	config, err := BuildConfigFromContext(kubeconfigPath, contextName)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig with context %s: %w", contextName, err)
	}

	baseHost := config.Host
	if idx := strings.Index(config.Host, "/clusters/"); idx > 0 {
		baseHost = config.Host[:idx]
	}

	config.Host = baseHost + "/clusters/root"

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	return &ClientManager{
		RestConfig:       config,
		Clientset:        clientset,
		DynamicClient:    dynamicClient,
		DiscoveryClient:  discoveryClient,
		baseHost:         baseHost,
		currentWorkspace: "root",
		discoveryCache:   make(map[string]interface{}),
	}, nil
}

func (c *ClientManager) SwitchWorkspace(path string) error {
	c.currentWorkspace = path
	c.RestConfig.Host = c.baseHost + "/clusters/" + path

	var err error
	c.DynamicClient, err = dynamic.NewForConfig(c.RestConfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client for workspace %s: %w", path, err)
	}

	c.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(c.RestConfig)
	if err != nil {
		return fmt.Errorf("failed to create discovery client for workspace %s: %w", path, err)
	}

	c.discoveryCache = make(map[string]interface{})
	return nil
}

func (c *ClientManager) SetWorkspace(path string) {
	c.SwitchWorkspace(path)
}

func (c *ClientManager) CurrentWorkspace() string {
	return c.currentWorkspace
}

func (c *ClientManager) BaseHost() string {
	return c.baseHost
}

func loadKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfigPath = home + "/.kube/config"
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

func GetContexts(kubeconfigPath string) ([]string, string, error) {
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, "", err
		}
		kubeconfigPath = home + "/.kube/config"
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfigPath

	config, err := loadingRules.Load()
	if err != nil {
		return nil, "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	contexts := make([]string, 0, len(config.Contexts))
	for name := range config.Contexts {
		contexts = append(contexts, name)
	}

	return contexts, config.CurrentContext, nil
}

func BuildConfigFromContext(kubeconfigPath, contextName string) (*rest.Config, error) {
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfigPath = home + "/.kube/config"
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfigPath

	overrides := &clientcmd.ConfigOverrides{
		ClusterInfo: api.Cluster{},
		Context:     api.Context{},
	}
	overrides.CurrentContext = contextName

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
}
