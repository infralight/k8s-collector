package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
)

const (
	DefaultNamespace     = "default"
	DefaultConfigMapName = "infralight-k8s-fetcher-config"
)

type Fetcher struct {
	log           *zerolog.Logger
	api           *kubernetes.Clientset
	namespace     string
	configMapName string
	config        *FetcherConfig
}

func NewFetcher(log *zerolog.Logger, api *kubernetes.Clientset) *Fetcher {
	return &Fetcher{
		log:           log,
		api:           api,
		namespace:     DefaultNamespace,
		configMapName: DefaultConfigMapName,
	}
}

func (f *Fetcher) SetNamespace(ns string) *Fetcher {
	f.namespace = ns
	return f
}

func (f *Fetcher) SetConfigMapName(name string) *Fetcher {
	f.configMapName = name
	return f
}

type fetchFunc struct {
	kind   string
	fn     func(context.Context) (items []interface{}, err error)
	onlyIf bool
}

func (f *Fetcher) Run(ctx context.Context, w io.Writer) error {
	// load our configuration from a ConfigMap
	err := f.loadConfig(ctx)
	if err != nil {
		if errors.IsNotFound(err) {
			// configuration doesn't exist, warn but do not fail, we'll use our
			// defaults
			f.log.Warn().
				Str("namespace", f.namespace).
				Str("config_map_name", f.configMapName).
				Msg("ConfigMap doesn't exist, using defaults")
		} else {
			return fmt.Errorf("failed loading configuration map: %w", err)
		}
	}

	enc := json.NewEncoder(w)

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
				Msg("Fetcher function failed")
			continue
		}

		if len(items) == 0 {
			continue
		}

		err = enc.Encode(map[string]interface{}{
			"kind":  fn.kind,
			"items": items,
		})
		if err != nil {
			// can't write JSON? consider this fatal
			f.log.Panic().
				Err(err).
				Str("kind", fn.kind).
				Msg("Failed writing data")
		}
	}

	return nil
}
