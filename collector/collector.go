package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/ido50/requests"
	"github.com/infralight/k8s-collector/collector/config"
	"github.com/infralight/k8s-collector/collector/filter"
	"github.com/infralight/k8s-collector/collector/k8stree"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"gopkg.in/mgo.v2/bson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	MaxItemSize = 1024 * 1500
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

	log            *zerolog.Logger
	client         *requests.HTTPClient
	dataCollectors []DataCollector
	dataFilters    []filter.DataFilter
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
		log:            conf.Log,
		clusterConfig:  clusterConfig,
		clusterID:      clusterID,
		dataCollectors: dataCollectors,
		dataFilters:    filter.All,
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

	f.log.Info().Str("Firefly Login Endpoint", f.conf.LoginEndpoint).Str("Firefly Endpoint", f.conf.Endpoint).Msg("Starting")

	// authenticate with the Infralight API
	if f.conf.DryRun {
		log.Info().Msg("Skipping authentication due to dry-run")
	} else {
		err = f.authenticate()
		if err != nil {
			return fmt.Errorf("failed authenticating with Infralight API: %w", err)
		}

		f.log.Info().Msg("Authenticated to Infralight App Server successfully")
	}

	var uniqueClusterId, fetchingId string

	if f.conf.DryRun {
		uniqueClusterId = "dry-run-cluster-id"
		fetchingId = "dry-run-fetching-id"
	} else {
		uniqueClusterId, err = f.getUniqueClusterId(ctx)
		if err != nil {
			return fmt.Errorf("failed finding Kubernetes unique cluster ID: %w", err)
		}

		fetchingId, err = f.startNewFetching(uniqueClusterId)
		if err != nil {
			return fmt.Errorf("failed starting new fetching with Infralight API: %w", err)
		}
	}

	log := f.log.With().
		Str("fetchingId", fetchingId).
		Str("uniqueClusterId", uniqueClusterId).
		Logger()

	log.Info().Msg("Starting new fetching process")

	fullData := make(map[string][]interface{}, len(f.dataCollectors))

	log.Debug().Int("amount", len(f.dataCollectors)).Msg("Running Kubernetes collectors")

	for _, dc := range f.dataCollectors {
		keyName, data, err := dc.Run(ctx, f.conf)
		if err != nil {
			if keyName == "helm_releases" {
				log.Warn().Err(err).Msg("Failed fetching helm releases")
				fullData[keyName] = data
				continue
			}
			return fmt.Errorf("%s collector failed: %w", dc.Source(), err)
		}

		fullData[keyName] = data
	}

	for _, filter := range f.dataFilters {
		log.Debug().Msg("Running filter")
		err := filter(ctx, fullData)
		if err != nil {
			log.Warn().Err(err).Msg("Filter failed")
			continue
		}
	}

	if f.conf.DryRun {
		enc := json.NewEncoder(os.Stdout)
		err = enc.Encode(fullData)
		if err != nil {
			return fmt.Errorf("failed encoding collected data: %w", err)
		}

		return nil
	}

	log.Debug().Msg("Sending data to Infralight App Server")

	err = f.sendHelmReleases(fetchingId, fullData["helm_releases"], fullData["k8s_types"])
	if err != nil {
		return fmt.Errorf("failed sending releases to Infralight: %w", err)
	}

	k8sTree, err := k8stree.GetK8sTree(fullData["k8s_objects"])
	if err != nil {
		return fmt.Errorf("failed getting k8s objects tree: %w", err)
	}

	err = f.sendK8sTree(fetchingId, k8sTree)
	if err != nil {
		return fmt.Errorf("failed sending k8s objects tree to Infralight: %w", err)
	}

	err = f.sendK8sObjects(fetchingId, fullData["k8s_objects"])
	if err != nil {
		return fmt.Errorf("failed sending objects to Infralight: %w", err)
	}

	return nil
}

func (f *Collector) authenticate() (err error) {
	var credentials struct {
		Token     string `json:"access_token"`
		ExpiresIn int64  `json:"expires_in"`
		Type      string `json:"token_type"`
	}

	err = requests.NewClient(f.conf.LoginEndpoint).
		NewRequest("POST", "/account/access_keys/login").
		JSONBody(map[string]interface{}{
			"accessKey": f.conf.AccessKey,
			"secretKey": f.conf.SecretKey,
		}).
		Into(&credentials).
		Run()
	if err != nil {
		return err
	}

	f.client = requests.NewClient(f.conf.Endpoint).
		Header("Authorization", fmt.Sprintf("Bearer %s", credentials.Token)).
		CompressWith(requests.CompressionAlgorithmGzip).
		ErrorHandler(func(httpStatus int, contentType string, body io.Reader) error {
			content, err := io.ReadAll(body)
			if err != nil {
				return fmt.Errorf("server returned unexpected status %d", httpStatus)
			}

			return fmt.Errorf("server returned %d: %q", httpStatus, content)
		})

	return nil
}

func (f *Collector) getUniqueClusterId(ctx context.Context) (clusterId string, err error) {
	kubeApi, err := kubernetes.NewForConfig(f.clusterConfig)
	if err != nil {
		return clusterId, fmt.Errorf("Failed creating Kubernetes Api object: %w", err)
	}

	kubeSystemNs, err := kubeApi.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	if err != nil {
		return clusterId, fmt.Errorf("Failed finding `kube-system` Kubernetes namespace: %w", err)
	}

	return string(kubeSystemNs.GetObjectMeta().GetUID()), nil
}

func (f *Collector) startNewFetching(clusterUniqueId string) (fetchingId string, err error) {
	fetchingId = bson.NewObjectId().Hex()
	req := f.client.
		NewRequest("HEAD", fmt.Sprintf("/integrations/k8s/%s/fetching", f.clusterID)).
		QueryParam("clusterUniqueId", clusterUniqueId).
		QueryParam("fetchingId", fetchingId).
		ExpectedStatus(http.StatusNoContent)
	if f.conf.OverrideUniqueClusterId {
		req.QueryParam("overrideUniqueClusterId", "1")
	}
	err = req.Run()
	return fetchingId, err
}

func (f *Collector) send(data map[string]interface{}) error {
	f.conf.Log.Debug().
		Interface("data", data).
		Msg("Sending collected data to Infralight")

	return f.client.
		NewRequest("POST", fmt.Sprintf("/integrations/k8s/%s/fetching", f.clusterID)).
		ExpectedStatus(http.StatusNoContent).
		JSONBody(data).
		Run()
}

func (f *Collector) sendK8sObjects(fetchingId string, data []interface{}) error {
	if len(data) == 0 {
		f.conf.Log.Warn().
			Str("FetchingId", fetchingId).
			Msg("No k8s objects to send to Infralight")
		return nil
	}
	f.conf.Log.Debug().
		Int("MessageSize", len(data)).
		Msg("Sending collected data to Infralight")

	totalBytes := 0
	var chunks [][]interface{}
	var objects []interface{}
	for idx, obj := range data {
		bytes, err := json.Marshal(obj)
		if err != nil {
			f.conf.Log.Err(err).
				Msg("failed to send resource")
		} else if len(bytes) > MaxItemSize {
			f.conf.Log.Warn().
				Msg("skipping massive resource")
		} else {
			totalBytes += len(bytes)
			objects = append(objects, obj)
		}
		if totalBytes > f.conf.PageSize*1000 || idx == len(data)-1 {
			chunks = append(chunks, objects)
			objects = []interface{}{}
			totalBytes = 0
		}
	}

	concurrentGoroutines := make(chan struct{}, f.conf.MaxGoRoutines)
	g, _ := errgroup.WithContext(context.Background())
	for _, chunkObjects := range chunks {
		concurrentGoroutines <- struct{}{}

		routineObjects := chunkObjects
		g.Go(func() error {
			defer func() {
				<-concurrentGoroutines
			}()
			body := make(map[string]interface{}, 2)
			body["fetchingId"] = fetchingId
			body["k8sObjects"] = routineObjects
			err := f.client.
				NewRequest(
					"POST",
					fmt.Sprintf("/integrations/k8s/%s/fetching/objects", f.clusterID),
				).
				ExpectedStatus(http.StatusNoContent).
				JSONBody(body).
				Run()
			if err != nil {
				log.Err(err).Str("ClusterId", f.clusterID).Str("FetchingId", fetchingId).
					Int("ResourcesInPage", len(routineObjects)).
					Msg("Error sending resources to server")
				return err
			}
			log.Info().Str("ClusterId", f.clusterID).Str("FetchingId", fetchingId).
				Int("ResourcesInPage", len(routineObjects)).
				Msg("Sent k8s objects page successfully")
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	err := f.client.
		NewRequest("PATCH", fmt.Sprintf("/integrations/k8s/%s/fetching", f.clusterID)).
		ExpectedStatus(http.StatusNoContent).
		JSONBody(map[string]interface{}{
			"fetchingId": fetchingId,
			"clusterId":  f.clusterID,
		}).
		Run()
	if err != nil {
		log.Err(err).
			Str("ClusterId", f.clusterID).
			Str("FetchingId", fetchingId).
			Msg("Error sending LOCK")
		return nil
	}
	log.Info().
		Str("ClusterId", f.clusterID).
		Str("FetchingId", fetchingId).
		Msg("Sent LOCK successfully")
	return nil
}

func (f *Collector) sendHelmReleases(
	fetchingId string,
	data []interface{},
	types []interface{},
) error {
	if len(data) == 0 {
		f.conf.Log.Warn().
			Str("FetchingId", fetchingId).
			Msg("No helm releases to send to Infralight")
		return nil
	}
	f.conf.Log.Debug().
		Str("FetchingId", fetchingId).
		Int("MessageSize", len(data)).
		Msg("Sending collected helm releases to Infralight")

	totalBytes := 0
	var chunks [][]interface{}
	var objects []interface{}
	for idx, obj := range data {
		bytes, _ := json.Marshal(obj)
		totalBytes += len(bytes)
		objects = append(objects, obj)

		if totalBytes > f.conf.PageSize*1000 || idx == len(data)-1 {
			chunks = append(chunks, objects)
			objects = []interface{}{}
			totalBytes = 0
		}
	}

	concurrentGoroutines := make(chan struct{}, f.conf.MaxGoRoutines)
	g, _ := errgroup.WithContext(context.Background())
	for _, chunkObjects := range chunks {
		concurrentGoroutines <- struct{}{}

		routineObjects := chunkObjects
		g.Go(func() error {
			defer func() {
				<-concurrentGoroutines
			}()
			body := make(map[string]interface{}, 3)
			body["fetchingId"] = fetchingId
			body["helmReleases"] = routineObjects
			body["k8sTypes"] = types
			err := f.client.
				NewRequest("POST", fmt.Sprintf("/integrations/k8s/%s/fetching/helm", f.clusterID)).
				ExpectedStatus(http.StatusNoContent).
				JSONBody(body).
				Run()
			if err != nil {
				log.Err(err).Str("ClusterId", f.clusterID).Str("FetchingId", fetchingId).
					Int("ResourcesInPage", len(routineObjects)).
					Msg("Error sending resources to server")
				return err
			}
			log.Info().Str("ClusterId", f.clusterID).Str("FetchingId", fetchingId).
				Int("ResourcesInPage", len(routineObjects)).
				Msg("Sent helm releases page successfully")
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	log.Info().
		Str("FetchingId", fetchingId).
		Int("Resources", len(data)).
		Msg("Sent all helm releases successfully")
	return nil
}

func (f *Collector) sendK8sTree(fetchingId string, data []k8stree.ObjectsTree) error {
	if len(data) == 0 {
		f.conf.Log.Warn().
			Str("FetchingId", fetchingId).
			Msg("No k8s objects trees to send to Infralight")
		return nil
	}
	f.conf.Log.Debug().
		Int("MessageSize", len(data)).
		Msg("Sending collected data to Infralight")

	totalBytes := 0
	var chunks [][]interface{}
	var objectsTrees []interface{}
	for idx, tree := range data {
		name := tree.Name
		tree.Name = ""
		bytes, err := json.Marshal(tree)
		if (tree.Children == nil || len(tree.Children) == 0) && tree.Kind != "Ingress" &&
			tree.Kind != "Provisioner" {
			f.conf.Log.Debug().
				Int("children", len(tree.Children)).
				Str("kind", tree.Kind).
				Str("name", name).
				Msg("skipping empty tree")
		} else if err != nil {
			f.conf.Log.Err(err).
				Int("children", len(tree.Children)).
				Str("kind", tree.Kind).
				Str("name", name).
				Msg("failed to send tree")
		} else if len(bytes) > MaxItemSize {
			f.conf.Log.Warn().
				Int("children", len(tree.Children)).
				Int("size", len(bytes)).
				Str("kind", tree.Kind).
				Str("name", name).
				Msg("skipping massive tree")
		} else {
			totalBytes += len(bytes)
			objectsTrees = append(objectsTrees, tree)
		}

		if totalBytes > f.conf.PageSize*1000 || idx == len(data)-1 {
			chunks = append(chunks, objectsTrees)
			objectsTrees = []interface{}{}
			totalBytes = 0
		}
	}

	concurrentGoroutines := make(chan struct{}, f.conf.MaxGoRoutines)
	g, _ := errgroup.WithContext(context.Background())
	for _, chunkObjectsTrees := range chunks {
		concurrentGoroutines <- struct{}{}

		routineObjects := chunkObjectsTrees
		g.Go(func() error {
			defer func() {
				<-concurrentGoroutines
			}()
			body := make(map[string]interface{}, 2)
			body["fetchingId"] = fetchingId
			body["k8sTrees"] = routineObjects
			err := f.client.
				NewRequest("POST", fmt.Sprintf("/integrations/k8s/%s/fetching/tree", f.clusterID)).
				ExpectedStatus(http.StatusNoContent).
				JSONBody(body).
				Run()
			if err != nil {
				log.Err(err).Str("ClusterId", f.clusterID).Str("FetchingId", fetchingId).
					Int("ResourcesInPage", len(routineObjects)).
					Msg("Error sending resources to server")
				return err
			}
			log.Info().Str("ClusterId", f.clusterID).Str("FetchingId", fetchingId).
				Int("ResourcesInPage", len(routineObjects)).
				Msg("Sent k8s objects trees page successfully")
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	log.Info().
		Str("FetchingId", fetchingId).
		Int("Resources", len(data)).
		Msg("Sent k8s objects trees page successfully")
	return nil
}
