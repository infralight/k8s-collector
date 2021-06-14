package collector

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"net/http"
	"regexp"

	"github.com/ido50/requests"
	"github.com/rs/zerolog/log"
	"gopkg.in/mgo.v2/bson"
	"k8s.io/client-go/rest"

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

	// Cluster configuration
	clusterConfig *rest.Config

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
	clusterConfig *rest.Config,
	conf *config.Config,
	dataCollectors ...DataCollector,
) *Collector {
	if conf == nil {
		panic("Configuration object must be provided")
	}

	return &Collector{
		conf:           conf,
		clusterConfig:  clusterConfig,
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
	log.Debug().Msg("Authenticating to Infralight App Server")
	// authenticate with the Infralight API
	f.accessToken, err = f.authenticate()
	if err != nil {
		return fmt.Errorf("failed authenticating with Infralight API: %w", err)
	}
	log.Info().Msg("Authenticated to Infralight App Server successfully")
	fetchingId, err := f.startNewFetching()
	if err != nil {
		return fmt.Errorf("failed starting new fetching with Infralight API: %w", err)
	}
	log.Info().Str("fetchingId", fetchingId).Msg("Starting new fetching process")
	fullData := make(map[string]interface{}, len(f.dataCollectors))
	log.Debug().Int("amount", len(f.dataCollectors)).Msg("Running Kubernetes collectors")
	for _, dc := range f.dataCollectors {
		keyName, data, err := dc.Run(ctx, f.conf)
		if err != nil {
			return fmt.Errorf("%s collector failed: %w", dc.Source(), err)
		}

		fullData[keyName] = data
	}
	fullData["fetchingId"] = fetchingId
	log.Debug().Msg("Sending data to Infralight App Server")
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

func (f *Collector) startNewFetching() (fetchingId string, err error) {
	fetchingId = bson.NewObjectId().Hex()
	var overrideMasterUrl string
	if f.conf.OverrideMasterUrl {
		overrideMasterUrl = "&overrideMasterUrl=1"
	}
	var masterUrl = b64.StdEncoding.EncodeToString([]byte(f.clusterConfig.Host))
	err = requests.NewClient(f.conf.Endpoint).
		Header("Authorization", fmt.Sprintf("Bearer %s", f.accessToken)).
		NewRequest("HEAD", fmt.Sprintf("/integrations/k8s/%s/fetching?masterUrl=%s&fetchingId=%s%s",
			f.clusterID, masterUrl, fetchingId, overrideMasterUrl)).
		CompressWith(requests.CompressionAlgorithmGzip).
		ExpectedStatus(http.StatusNoContent).
		Run()
	return fetchingId, err
}

func (f *Collector) send(data map[string]interface{}) error {
	f.conf.Log.Debug().
		Interface("data", data).
		Msg("Sending collected data to Infralight")

	return requests.NewClient(f.conf.Endpoint).
		Header("Authorization", fmt.Sprintf("Bearer %s", f.accessToken)).
		NewRequest("POST", fmt.Sprintf("/integrations/k8s/%s/fetching", f.clusterID)).
		CompressWith(requests.CompressionAlgorithmGzip).
		ExpectedStatus(http.StatusNoContent).
		JSONBody(data).
		Run()
}
