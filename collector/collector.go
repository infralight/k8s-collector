package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/DataDog/zstd"
	"github.com/ido50/requests"
	"github.com/rs/zerolog"
	"k8s.io/client-go/kubernetes"
)

type Collector struct {
	log           *zerolog.Logger
	api           kubernetes.Interface
	namespace     string
	configMapName string
	config        *CollectorConfig
}

type CollectorData struct {
	Objects []interface{} `json:"objects"`
}

func NewCollector(log *zerolog.Logger, api kubernetes.Interface) *Collector {
	return &Collector{
		log:           log,
		api:           api,
		namespace:     DefaultNamespace,
		configMapName: DefaultConfigMapName,
	}
}

func (f *Collector) SetNamespace(ns string) *Collector {
	f.namespace = ns
	return f
}

func (f *Collector) SetConfigMapName(name string) *Collector {
	f.configMapName = name
	return f
}

type fetchFunc struct {
	kind   string
	fn     func(context.Context) (items []interface{}, err error)
	onlyIf bool
}

func (f *Collector) Run(ctx context.Context) error {
	// load our configuration from a ConfigMap
	err := f.loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed loading configuration map: %w", err)
	}

	var objects []interface{} // nolint: prealloc

	for _, fn := range []fetchFunc{
		{"ClusterRole", f.getClusterRoles, f.config.FetchClusterRoles},
		{"ConfigMap", f.getConfigMaps, f.config.FetchConfigMaps},
		{"CronJob", f.getCronJobs, f.config.FetchCronJobs},
		{"Event", f.getEvents, f.config.FetchEvents},
		{"DaemonSet", f.getDaemonSets, f.config.FetchDaemonSets},
		{"Deployment", f.getDeployments, f.config.FetchDeployments},
		{"Ingress", f.getIngresses, f.config.FetchIngresses},
		{"Job", f.getJobs, f.config.FetchJobs},
		{"Namespace", f.getNamespaces, f.config.FetchNamespaces},
		{"Node", f.getNodes, f.config.FetchNodes},
		{"ReplicaSet", f.getReplicaSets, f.config.FetchReplicaSets},
		{"ReplicationController", f.getReplicationControllers, f.config.FetchReplicationControllers},
		{"ServiceAccount", f.getServiceAccounts, f.config.FetchServiceAccounts},
		{"Service", f.getServices, f.config.FetchServices},
		{"Secret", f.getSecrets, f.config.FetchSecrets},
		{"StatefulSet", f.getStatefulSet, f.config.FetchStatefulSets},
		{"PersistentVolumeClaim", f.getPersistentVolumeClaims, f.config.FetchPersistentVolumeClaims},
		{"PersistentVolume", f.getPersistentVolumes, f.config.FetchPersistentVolumes},
		{"Pod", f.getPods, f.config.FetchPods},
	} {
		if !fn.onlyIf {
			continue
		}

		items, err := fn.fn(ctx)
		if err != nil {
			f.log.Warn().
				Err(err).
				Str("kind", fn.kind).
				Msg("Collector function failed")
			continue
		}

		if len(items) == 0 {
			continue
		}

		objects = append(objects, items...)
	}

	// start building an HTTP request to send the data to the Infralight API
	req := requests.NewClient(f.config.Endpoint).
		Header("Authorization", fmt.Sprintf("Bearer %s", f.config.APIKey)).
		NewRequest("POST", "").
		ExpectedStatus(http.StatusNoContent)
	if err != nil {
		return fmt.Errorf("failed sending data to %s: %w", f.config.Endpoint, err)
	}

	// encode data to JSON
	rawBody, err := json.Marshal(CollectorData{objects})
	if err != nil {
		return fmt.Errorf("failed encoding to JSON: %w", err)
	}

	f.log.Debug().
		Bytes("body", rawBody).
		Msg("Sending data to Infralight")

	// compress data with zstd. if this fails, we'll still send, but uncompressed
	compressedBody, err := zstd.CompressLevel(nil, rawBody, 10)
	if err != nil {
		f.log.Warn().
			Err(err).
			Msg("Failed compressing data, will send uncompressed")

		req.Body(rawBody, "application/json")
	} else {
		req.
			Header("Content-Encoding", "zstd").
			Body(compressedBody, "application/json")
	}

	// send the request, we will fail unless the server returned the expected
	// status (204 no content)
	err = req.Run()
	if err != nil {
		return fmt.Errorf("failed sending data to Infralight: %w", err)
	}

	return nil
}
