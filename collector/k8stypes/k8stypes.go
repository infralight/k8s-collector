package k8stypes

import (
	"context"
	"fmt"
	"github.com/infralight/k8s-collector/collector/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Collector is a struct implementing the DataCollector interface. It wraps a
// Kubernetes API client object.
type Collector struct {
	// client object for the Kubernetes API server
	api kubernetes.Interface
}

// New creates a new instance of the Collector struct. A Kubernetes API client
// object must be provided. This can either be a client for a real API server,
// a fake client from k8s.io/client-go/kubernetes/fake, or any object that
// implements the kubernetes.Interface interface.
func New(api kubernetes.Interface) *Collector {
	return &Collector{
		api: api,
	}
}

// DefaultConfiguration creates a Collector instance with default configuration
// to connect to a local Kubernetes API Server. When running outside of the
// Kubernetes cluster, the path to the kubeconfig file must be provided. If
// empty, the default in-cluster configuration is used.
func DefaultConfiguration(external string) (
	collector *Collector,
	err error,
) {
	// Load configuration for the Kubernetes API client. We are either running
	// from inside the cluster (i.e. inside a pod) or outside of the cluster.
	var apiConfig *rest.Config
	if external != "" {
		apiConfig, err = clientcmd.BuildConfigFromFlags("", external)
	} else {
		// Load configuration to connect to the Kubernetes API from within a K8s
		// cluster
		apiConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return collector, fmt.Errorf("failed loading Kubernetes configuration: %w", err)
	}

	// Create a new instance of the Kubernetes API client
	api, err := kubernetes.NewForConfig(apiConfig)
	if err != nil {
		return collector, fmt.Errorf("failed getting K8s client set: %w", err)
	}

	return New(api), nil
}

// Source is required by the DataCollector interface to return a name for the
// collector's source, in this case the K8s API Server.
func (f *Collector) Source() string {
	return "K8s API Server"
}

// Run executes the collector with the provided configuration object, and
// returns a list of supported resources from the cluster.
func (f *Collector) Run(ctx context.Context, conf *config.Config) (
	keyName string,
	types []interface{},
	err error,
) {

	var supportedResources []map[string]interface{}

	apiGroups, err := f.api.Discovery().ServerPreferredResources()
	if err != nil {
		return "", nil, err
	}
	for _, apiGroup := range apiGroups {
		if len(apiGroup.APIResources) == 0 {
			continue
		}
		for _, resource := range apiGroup.APIResources {
			var resourceConf = make(map[string]interface{})
			resourceConf["kind"] = resource.Kind
			resourceConf["namespaced"] = resource.Namespaced
			resourceConf["apiVersion"] = apiGroup.GroupVersion
			supportedResources = append(supportedResources, resourceConf)
		}
	}

	types = make([]interface{}, len(supportedResources))
	for i, rel := range supportedResources {
		types[i] = rel
	}

	return "k8s_types", types, nil
}
