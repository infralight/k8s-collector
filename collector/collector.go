package collector

import (
	"context"
	"fmt"
	"net/http"

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
	accessToken   string
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

	// authenticate with the Infralight API
	f.accessToken, err = f.authenticate()
	if err != nil {
		return fmt.Errorf("failed authenticating with Infralight API: %w", err)
	}

	objects := f.collect(ctx)

	err = f.send(objects)
	if err != nil {
		return fmt.Errorf("failed sending objects to Infralight: %w", err)
	}

	return nil
}

func (f *Collector) authenticate() (accessToken string, err error) {
	var credentials struct {
		Token     string `json:"access_token"`
		ExpiresIn int64  `json:"expires_in"`
		Type      string `json:"token_type"`
	}

	err = requests.NewClient(f.config.Endpoint).
		NewRequest("POST", "/sink/login").
		JSONBody(map[string]interface{}{
			"accessKey": f.config.AccessKey,
			"secretKey": f.config.SecretKey,
		}).
		Into(&credentials).
		Run()
	return credentials.Token, err
}

func (f *Collector) collect(ctx context.Context) (objects []interface{}) {
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

	return objects
}

func (f *Collector) send(objects []interface{}) error {
	return requests.NewClient(f.config.Endpoint).
		Header("Authorization", fmt.Sprintf("Bearer %s", f.accessToken)).
		NewRequest("POST", "/sink/send").
		CompressWith(requests.CompressionAlgorithmGzip).
		ExpectedStatus(http.StatusNoContent).
		JSONBody(CollectorData{objects}).
		Run()
}
