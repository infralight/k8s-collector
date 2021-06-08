package collector

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/ido50/requests"
	"github.com/infralight/k8s-collector/collector/config"
)

// DataCollector is an interface for objects that collect data from K8s-related
// components such as the Kubernetes API Server or Helm
type DataCollector interface {
	// Source is a method that returns a unique name for the collector source
	Source() string

	// Run executes the data collector. The configuration object is always passed
	// and is never empty or nil. Every collector must return a name for the
	// key under which the data will be sent to the Infralight App Server, the
	// data itself (which is a list of arbitrary objects), and an optional error.
	Run(context.Context, *config.Config) (
		keyName string,
		data []interface{},
		err error,
	)
}

// Collector is an execution-scoped object encapsulating the entire collection
// process.
type Collector struct {
	// the JWT access token used to authenticate with the Infralight App server.
	// this is automatically generated
	accessToken string

	// the unique identifier of the cluster we're collecting data from (must be
	// provided externally)
	clusterID string

	// the collector's configuration
	conf *config.Config

	// the data collectors
	dataCollectors []DataCollector
}

var clusterIDRegex = regexp.MustCompile(`^[a-z0-9-_]+$`)

// New creates a new instance of the Collector struct. A Kubernetes cluster ID
// must be provided, together with a configuration object and a list of objects
// implementing the DataCollector interface.
//
// The cluster ID is a string of alphanumeric characters, dashes and underscores,
// of any length. Spaces are not allowed.
//
// A configuration object must be provided.
func New(
	clusterID string,
	conf *config.Config,
	dataCollectors ...DataCollector,
) *Collector {
	if conf == nil {
		panic("Configuration object must be provided")
	}

	return &Collector{
		conf:           conf,
		clusterID:      clusterID,
		dataCollectors: dataCollectors,
	}
}

// Run executes the collector. The process includes authentication with the
// Infralight App Server, execution of all data collectors, and sending of the
// data to the App Server for storage.
func (f *Collector) Run(ctx context.Context) (err error) {
	// verify cluster ID is valid
	if !clusterIDRegex.MatchString(f.clusterID) {
		return fmt.Errorf("invalid cluster ID, must match %s", clusterIDRegex)
	}

	// authenticate with the Infralight API
	f.accessToken, err = f.authenticate()
	if err != nil {
		return fmt.Errorf("failed authenticating with Infralight API: %w", err)
	}

	fullData := make(map[string]interface{}, len(f.dataCollectors))
	for _, dc := range f.dataCollectors {
		keyName, data, err := dc.Run(ctx, f.conf)
		if err != nil {
			return fmt.Errorf("%s collector failed: %w", dc.Source(), err)
		}

		fullData[keyName] = data
	}

	err = f.send(fullData)
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

	err = requests.NewClient(f.conf.Endpoint).
		NewRequest("POST", "/account/access_keys/login").
		JSONBody(map[string]interface{}{
			"accessKey": f.conf.AccessKey,
			"secretKey": f.conf.SecretKey,
		}).
		Into(&credentials).
		Run()
	return credentials.Token, err
}

func (f *Collector) send(data map[string]interface{}) error {
	return requests.NewClient(f.conf.Endpoint).
		Header("Authorization", fmt.Sprintf("Bearer %s", f.accessToken)).
		NewRequest("PUT", fmt.Sprintf("/integrations/k8s/%s", f.clusterID)).
		CompressWith(requests.CompressionAlgorithmGzip).
		ExpectedStatus(http.StatusNoContent).
		JSONBody(data).
		Run()
}
