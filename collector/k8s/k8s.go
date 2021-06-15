package k8s

import (
	"context"
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
type Object struct {
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
	var numFailed int
	var lastErr error

	type fetchFunc struct {
		kind   string
		fn     func(context.Context, *config.Config) (items []interface{}, err error)
		onlyIf bool
	}

	funcs := []fetchFunc{
		{"ClusterRole", f.getClusterRoles, conf.FetchClusterRoles},
		{"ConfigMap", f.getConfigMaps, conf.FetchConfigMaps},
		{"CronJob", f.getCronJobs, conf.FetchCronJobs},
		{"Event", f.getEvents, conf.FetchEvents},
		{"DaemonSet", f.getDaemonSets, conf.FetchDaemonSets},
		{"Deployment", f.getDeployments, conf.FetchDeployments},
		{"Ingress", f.getIngresses, conf.FetchIngresses},
		{"Job", f.getJobs, conf.FetchJobs},
		{"Namespace", f.getNamespaces, conf.FetchNamespaces},
		{"Node", f.getNodes, conf.FetchNodes},
		{"ReplicaSet", f.getReplicaSets, conf.FetchReplicaSets},
		{"ReplicationController", f.getReplicationControllers, conf.FetchReplicationControllers},
		{"ServiceAccount", f.getServiceAccounts, conf.FetchServiceAccounts},
		{"Service", f.getServices, conf.FetchServices},
		{"Secret", f.getSecrets, conf.FetchSecrets},
		{"StatefulSet", f.getStatefulSet, conf.FetchStatefulSets},
		{"PersistentVolumeClaim", f.getPersistentVolumeClaims, conf.FetchPersistentVolumeClaims},
		{"PersistentVolume", f.getPersistentVolumes, conf.FetchPersistentVolumes},
		{"Pod", f.getPods, conf.FetchPods},
	}

	for _, fn := range funcs {
		if !fn.onlyIf {
			continue
		}

		items, err := fn.fn(ctx, conf)
		if err != nil {
			numFailed++
			lastErr = fmt.Errorf("failed collecting %s: %w", fn.kind, err)
			continue
		}

		if len(items) == 0 {
			continue
		}

		for _, item := range items {
			objects = append(objects, Object{
				Kind:   fn.kind,
				Object: item,
			})
		}
	}

	if numFailed == len(funcs) {
		return "k8s_objects", objects, lastErr
	}

	log.Info().Int("amount", len(objects)).Msg("Finished collecting Kubernetes objects")

	return "k8s_objects", objects, nil
}
