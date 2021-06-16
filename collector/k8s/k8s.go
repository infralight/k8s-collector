package k8s

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/infralight/k8s-collector/collector/config"
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
func DefaultConfiguration(apiConfig *rest.Config) (
	collector *Collector,
	err error,
) {
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

// Object is a pointless struct type that we have no choice but create due to
// an issue with how the official Kubernetes client encodes objects to JSON.
// The "Kind" attribute that each object has is in an embedded struct that is
// set with the following struct tag: json:",inline". The problem is that the
// "inline" struct tag is still in proposal status and not supported by Go,
// (see here: https://github.com/golang/go/issues/6213), and so JSON objects are
// missing the "kind" attribute. This is just a workaround to ensure we also
// send the kind.

type KubernetesObject struct {
	Kind   string      `json:"kind"`
	Object interface{} `json:"object"`
}

// Run executes the collector with the provided configuration object, and
// returns a list of collected objects from the Kubernetes cluster.
func (f *Collector) Run(ctx context.Context, conf *config.Config) (
	keyName string,
	objects []interface{},
	err error,
) {
	log.Debug().Msg("Starting collect Kubernetes objects")

	allowList := map[string]bool{
		"ClusterRole":           conf.FetchClusterRoles,
		"ConfigMap":             conf.FetchConfigMaps,
		"CronJob":               conf.FetchCronJobs,
		"Event":                 conf.FetchEvents,
		"DaemonSet":             conf.FetchDaemonSets,
		"Deployment":            conf.FetchDeployments,
		"Ingress":               conf.FetchIngresses,
		"Job":                   conf.FetchJobs,
		"Namespace":             conf.FetchNamespaces,
		"Node":                  conf.FetchNodes,
		"ReplicaSet":            conf.FetchReplicaSets,
		"ReplicationController": conf.FetchReplicationControllers,
		"ServiceAccount":        conf.FetchServiceAccounts,
		"Service":               conf.FetchServices,
		"Secret":                conf.FetchSecrets,
		"StatefulSet":           conf.FetchStatefulSets,
		"PersistentVolumeClaim": conf.FetchPersistentVolumeClaims,
		"PersistentVolume":      conf.FetchPersistentVolumes,
		"Pod":                   conf.FetchPods,
	}

	apiResourcesList, err := f.api.Discovery().ServerPreferredResources()

	for _, apiResource := range apiResourcesList {
		for _, resource := range apiResource.APIResources {
			var uri string
			if apiResource.GroupVersion == "v1" && apiResource.APIVersion == "" {
				// The URL for for api v1 is different from the external apis
				uri = "api/v1"
			} else {
				uri = fmt.Sprintf("apis/%s", apiResource.GroupVersion)
			}
			toFetch, ok := allowList[resource.Kind]
			if toFetch || !ok {
				itemsResponse := f.api.Discovery().RESTClient().Get().RequestURI(uri).Resource(resource.Name).Do(ctx)
				var responseCode int
				itemsResponse.StatusCode(&responseCode)
				if responseCode != 200 {
					log.Err(itemsResponse.Error()).Str("ApiVersion", uri).Str("kind", resource.Kind).Msg("Error receiving response while listing resources")
					continue
				}
				type ResourcesListResponse struct {
					Kind       string                   `json:"kind"`
					APIVersion string                   `json:"apiVersion"`
					Items      []map[string]interface{} `json:"items"`
				}
				var itemsDict = ResourcesListResponse{}
				responseData, err := itemsResponse.Raw()
				if err != nil {
					log.Err(err).Str("ApiVersion", uri).Str("kind", resource.Kind).Msg("Error reading response while listing resources")
				}
				json.Unmarshal(responseData, &itemsDict)
				for _, item := range itemsDict.Items {
					item["apiVersion"] = apiResource.GroupVersion
					item["Kind"] = resource.Kind
					objects = append(objects, KubernetesObject{
						Kind:   resource.Kind,
						Object: item,
					})
				}
				log.Debug().Int("items", len(itemsDict.Items)).Str("ApiVersion", uri).Str("kind", resource.Kind).Msg("Found items for resource")
			} else {
				log.Warn().Str("ApiVersion", uri).Str("kind", resource.Kind).Msg("Ignoring resources due to policy")
			}
		}
	}
	log.Info().Int("items", len(objects)).Int("apis", len(apiResourcesList)).Msg("Finished Kubernetes cluster fetching")
	return "k8s_objects", objects, nil
}
