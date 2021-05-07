package collector

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"regexp"

	"github.com/ido50/requests"
	"github.com/rs/zerolog"
	"k8s.io/client-go/kubernetes"
)

type Collector struct {
	log *zerolog.Logger

	// client object for the Kubernetes API server
	api kubernetes.Interface

	// the JWT access token used to authenticate with the Infralight App server.
	// this is automatically generated
	accessToken string

	// the unique identifier of the cluster we're collecting data from (must be
	// provided externally)
	clusterID string

	// file system object from which configuration files are read. by default,
	// this is the local file system; an in-memory file system is used in the
	// unit tests
	fs fs.FS

	// the directory inside fs where configuration files are stored. by default,
	// this is /etc/config
	configDir string

	// the collector's configuration
	config *CollectorConfig
}

// K8sObject is a pointless struct type that we have no choice but create due to
// an issue with how the official Kubernetes client encodes objects to JSON.
// The "Kind" attribute that each object has is in an embedded struct that is
// set with the following struct tag: json:",inline". The problem is that the
// "inline" struct tag is still in proposal status and not supported by Go,
// (see here: https://github.com/golang/go/issues/6213), and so JSON objects are
// missing the "kind" attribute. This is just a workaround to ensure we also
// send the kind.
type K8sObject struct {
	Kind   string      `json:"kind"`
	Object interface{} `json:"object"`
}

type CollectorData struct {
	ClusterID string      `json:"cluster_id"`
	Objects   []K8sObject `json:"objects"`
}

var clusterIDRegex = regexp.MustCompile(`^[a-z0-9-_]+$`)

func NewCollector(
	clusterID string,
	log *zerolog.Logger,
	api kubernetes.Interface,
) *Collector {
	return &Collector{
		log:       log,
		api:       api,
		clusterID: clusterID,
		fs:        &localFS{},
		configDir: "/etc/config",
	}
}

func (f *Collector) SetFS(fs fs.FS) *Collector {
	f.fs = fs
	return f
}

func (f *Collector) SetConfigDir(dir string) *Collector {
	f.configDir = dir
	return f
}

type fetchFunc struct {
	kind   string
	fn     func(context.Context) (items []interface{}, err error)
	onlyIf bool
}

func (f *Collector) Run(ctx context.Context) error {
	// verify cluster ID is valid
	if !clusterIDRegex.MatchString(f.clusterID) {
		return fmt.Errorf("invalid cluster ID, must match %s", clusterIDRegex)
	}

	// load our configuration from a ConfigMap
	err := f.loadConfig()
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
		NewRequest("POST", "/account/access_keys/login").
		JSONBody(map[string]interface{}{
			"accessKey": f.config.AccessKey,
			"secretKey": f.config.SecretKey,
		}).
		Into(&credentials).
		Run()
	return credentials.Token, err
}

func (f *Collector) collect(ctx context.Context) (objects []K8sObject) {
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

		for _, item := range items {
			objects = append(objects, K8sObject{
				Kind:   fn.kind,
				Object: item,
			})
		}
	}

	return objects
}

func (f *Collector) send(objects []K8sObject) error {
	return requests.NewClient(f.config.Endpoint).
		Header("Authorization", fmt.Sprintf("Bearer %s", f.accessToken)).
		NewRequest("PUT", fmt.Sprintf("/integrations/k8s/%s", f.clusterID)).
		CompressWith(requests.CompressionAlgorithmGzip).
		ExpectedStatus(http.StatusNoContent).
		JSONBody(CollectorData{
			Objects: objects,
		}).
		Run()
}

type localFS struct{}

func (fs *localFS) Open(name string) (fs.File, error) {
	return os.Open("/" + name)
}
