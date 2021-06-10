package helm

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/infralight/k8s-collector/collector/config"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

// Collector is a struct implementing the DataCollector interface. It wraps a
// Helm SDK configuration object.
type Collector struct {
	sdkConfig *action.Configuration
}

// New creates a new instance of the Collector struct. A Helm SDK configuration
// object must be provided.
func New(sdkConfig *action.Configuration) *Collector {
	return &Collector{
		sdkConfig: sdkConfig,
	}
}

// DefaultConfiguration creates a Collector instance with default configuration
// to use Helm on a local Kubernetes cluster. A Printf-like function can be
// provided so the SDK uses an application-wide logger object. If nil, the
// `log.Printf` function from the standard library is used. The HELM_DRIVER
// environment variable is also taken into account, as described in the Helm
// docs: https://helm.sh/docs/topics/advanced/#storage-backends
func DefaultConfiguration(pf action.DebugLog) (c *Collector, err error) {
	settings := cli.New()

	conf := new(action.Configuration)

	if pf == nil {
		pf = log.Printf
	}

	err = conf.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), pf)
	if err != nil {
		return nil, fmt.Errorf("failed loading default Helm configuration: %w", err)
	}

	return New(conf), nil
}

// Source is required by the DataCollector interface to return a name for the
// collector's source, in this case the Helm SDK.
func (c *Collector) Source() string {
	return "Helm SDK"
}

// Run executes the collector with the provided configuration object, and
// returns a list of collected Helm releases from the Kubernetes cluster.
func (c *Collector) Run(ctx context.Context, _ *config.Config) (
	keyName string,
	data []interface{},
	err error,
) {
	client := action.NewList(c.sdkConfig)
	client.Deployed = true

	results, err := client.Run()
	if err != nil {
		return keyName, data, fmt.Errorf("list failed: %w", err)
	}

	releases := make([]interface{}, len(results))
	for i, rel := range results {
		releases[i] = rel
	}

	return "helm_releases", releases, nil
}
