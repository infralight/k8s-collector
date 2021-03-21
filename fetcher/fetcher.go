package fetcher

import (
	"context"
	"encoding/json"
	"io"

	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const ConfigMapName = "infralight-fetcher-config"

type Fetcher struct {
	log *zerolog.Logger
	api *kubernetes.Clientset
}

func NewFetcher(log *zerolog.Logger, api *kubernetes.Clientset) *Fetcher {
	return &Fetcher{
		log: log,
		api: api,
	}
}

type fetchFunc struct {
	name string
	fn   func(context.Context) (interface{}, error)
}

func (f *Fetcher) Run(ctx context.Context, w io.Writer) error {
	enc := json.NewEncoder(w)

	for _, fn := range []fetchFunc{
		{"nodes", f.getNodes},
		{"namespaces", f.getNamespaces},
		{"config_maps", f.getConfigMaps},
		{"replication_controllers", f.getReplicationControllers},
		{"events", f.getEvents},
		{"services", f.getServices},
		{"service_accounts", f.getServiceAccounts},
		{"pods", f.getPods},
		{"persistent_volumes", f.getPersistentVolumes},
		{"persistent_volume_claims", f.getPersistentVolumeClaims},
	} {
		data, err := fn.fn(ctx)
		if err != nil {
			f.log.Warn().
				Err(err).
				Str("func", fn.name).
				Msg("Fetcher function failed")
			continue
		}

		err = enc.Encode(map[string]interface{}{fn.name: data})
		if err != nil {
			// can't write JSON? consider this fatal
			f.log.Panic().
				Err(err).
				Str("func", fn.name).
				Msg("Failed writing data")
		}
	}

	return nil
}

func (f *Fetcher) getNamespaces(ctx context.Context) (data interface{}, err error) {
	return f.api.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
}

func (f *Fetcher) getPods(ctx context.Context) (data interface{}, err error) {
	return f.api.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
}

func (f *Fetcher) getEvents(ctx context.Context) (data interface{}, err error) {
	return f.api.CoreV1().Events("").List(ctx, metav1.ListOptions{})
}

func (f *Fetcher) getConfigMaps(ctx context.Context) (data interface{}, err error) {
	return f.api.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{})
}

func (f *Fetcher) getReplicationControllers(ctx context.Context) (data interface{}, err error) {
	return f.api.CoreV1().ReplicationControllers("").List(ctx, metav1.ListOptions{})
}

func (f *Fetcher) getNodes(ctx context.Context) (data interface{}, err error) {
	return f.api.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
}

func (f *Fetcher) getServices(ctx context.Context) (data interface{}, err error) {
	return f.api.CoreV1().Services("").List(ctx, metav1.ListOptions{})
}

func (f *Fetcher) getServiceAccounts(ctx context.Context) (data interface{}, err error) {
	return f.api.CoreV1().ServiceAccounts("").List(ctx, metav1.ListOptions{})
}

func (f *Fetcher) getPersistentVolumes(ctx context.Context) (data interface{}, err error) {
	return f.api.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
}

func (f *Fetcher) getPersistentVolumeClaims(ctx context.Context) (data interface{}, err error) {
	return f.api.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
}
