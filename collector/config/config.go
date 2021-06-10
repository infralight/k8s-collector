package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

const (
	// AccessKeyEnvVar is the name of the environment variable where the access
	// key to the Infralight App Server must be provided
	AccessKeyEnvVar = "INFRALIGHT_ACCESS_KEY"

	// SecretKeyEnvVar is the name of the environment variable where the secret
	// key to the Infralight App Server must be provided
	SecretKeyEnvVar = "INFRALIGHT_SECRET_KEY" // nolint: gosec

	// DefaultConfigDir is the path to the default directory where configuration
	// files (generally mounted from a Kubernetes ConfigMap) must be present.
	DefaultConfigDir = "/etc/config"
)

var (
	// ErrAccessKeys is an error returned when the environment variables for the
	// access and secret keys are not provided or empty.
	ErrAccessKeys = errors.New("access and secret keys must be provided")

	// ErrEndpoint is an error returned when the configuration directory is
	// missing an endpoint setting (endpoint is the URL to the Infralight App
	// Server).
	ErrEndpoint = errors.New("Infralight endpoint must be provided")
)

// Config represents configuration to the collector library. It is shared
// between the different data collectors (impementing the collector.DataCollector
// interface).
type Config struct {
	// File system object from which configuration files are read. by default,
	// this is the local file system; an in-memory file system is used in the
	// unit tests
	FS fs.FS

	// The directory inside fs where configuration files are stored. by default,
	// this is /etc/config
	ConfigDir string

	// The logger instance
	Log *zerolog.Logger

	// AccessKey is the Infralight access key
	AccessKey string

	// SecretKey is the Infralight secret key
	SecretKey string

	// Endpoint is the URL to the Infralight App Server
	Endpoint string

	// Namespace is the Kubernets namespace we're collecting data from (if empty,
	// all namespaces are collected)
	Namespace string

	// IgnoreNamespaces is a list of namespaces to ignore (only taken into
	// account when Namespace is empty)
	IgnoreNamespaces []string

	// FetchEvents is a boolean indicating whether to collect Kubernetes Events
	FetchEvents bool
	// FetchConfigMaps is a boolean indicating whether to collect Kubernetes ConfigMaps
	FetchConfigMaps bool
	// FetchReplicationControllers is a boolean indicating whether to collect Kubernetes ReplicationControllers
	FetchReplicationControllers bool
	// FetchSecrets is a boolean indicating whether to collect Kubernetes Secrets
	FetchSecrets bool
	// FetchServices is a boolean indicating whether to collect Kubernetes Services
	FetchServices bool
	// FetchServiceAccounts is a boolean indicating whether to collect Kubernetes ServiceAccounts
	FetchServiceAccounts bool
	// FetchPods is a boolean indicating whether to collect Kubernetes Pods
	FetchPods bool
	// FetchNodes is a boolean indicating whether to collect Kubernetes Nodes
	FetchNodes bool
	// FetchPersistentVolumes is a boolean indicating whether to collect Kubernetes PersistentVolumes
	FetchPersistentVolumes bool
	// FetchPersistentVolumeClaims is a boolean indicating whether to collect Kubernetes PersistentVolumeClaims
	FetchPersistentVolumeClaims bool
	// FetchNamespaces is a boolean indicating whether to collect Kubernetes Namespaces
	FetchNamespaces bool
	// FetchDeployments is a boolean indicating whether to collect Kubernetes Deployments
	FetchDeployments bool
	// FetchDaemonSets is a boolean indicating whether to collect Kubernetes DaemonSets
	FetchDaemonSets bool
	// FetchReplicaSets is a boolean indicating whether to collect Kubernetes ReplicaSets
	FetchReplicaSets bool
	// FetchStatefulSets is a boolean indicating whether to collect Kubernetes StatefulSets
	FetchStatefulSets bool
	// FetchJobs is a boolean indicating whether to collect Kubernetes Jobs
	FetchJobs bool
	// FetchCronJobs is a boolean indicating whether to collect Kubernetes CronJobs
	FetchCronJobs bool
	// FetchIngresses is a boolean indicating whether to collect Kubernetes Ingresses
	FetchIngresses bool
	// FetchClusterRoles is a boolean indicating whether to collect Kubernetes ClusterRoles
	FetchClusterRoles bool
}

// LoadConfig creates a new configuration object. A logger object, a file-system
// object (where configuration files are stored), and a path to the configuration
// directory may be provided. All parameters are optional. If not provided,
// a noop logger is used, the local file system is used, and DefaultConfigDir is
// used.
func LoadConfig(
	log *zerolog.Logger,
	cfs fs.FS,
	configDir string,
) (conf *Config, err error) {
	if log == nil {
		l := zerolog.Nop()
		log = &l
	}
	if cfs == nil {
		log.Debug().Msg("No file system object provided, using default one")
		cfs = &localFS{}
	}

	if configDir == "" {
		configDir = DefaultConfigDir
	}

	// load Infralight API Key from the environment, this is required
	accessKey := os.Getenv(AccessKeyEnvVar)
	secretKey := os.Getenv(SecretKeyEnvVar)
	if accessKey == "" || secretKey == "" {
		return conf, ErrAccessKeys
	}

	conf = &Config{
		FS:        cfs,
		ConfigDir: configDir,
		Log:       log,
	}

	conf.Endpoint = strings.TrimSuffix(parseOne(conf.etcConfig("endpoint"), ""), "/")
	if conf.Endpoint == "" {
		return conf, ErrEndpoint
	}

	conf.AccessKey = accessKey
	conf.SecretKey = secretKey
	conf.Namespace = parseOne(conf.etcConfig("collector.watchNamespace"), "")
	conf.IgnoreNamespaces = parseMultiple(conf.etcConfig("collector.ignoreNamespaces"), nil)
	conf.FetchEvents = parseBool(conf.etcConfig("collector.resources.events"), true)
	conf.FetchReplicationControllers = parseBool(conf.etcConfig("collector.resources.replicationControllers"), true)
	conf.FetchServices = parseBool(conf.etcConfig("collector.resources.services"), true)
	conf.FetchServiceAccounts = parseBool(conf.etcConfig("collector.resources.serviceAccounts"), true)
	conf.FetchPods = parseBool(conf.etcConfig("collector.resources.pods"), true)
	conf.FetchNodes = parseBool(conf.etcConfig("collector.resources.nodes"), true)
	conf.FetchPersistentVolumes = parseBool(conf.etcConfig("collector.resources.persistentVolumes"), true)
	conf.FetchPersistentVolumeClaims = parseBool(conf.etcConfig("collector.resources.persistentVolumeClaims"), true)
	conf.FetchNamespaces = parseBool(conf.etcConfig("collector.resources.namespaces"), true)
	conf.FetchConfigMaps = parseBool(conf.etcConfig("collector.resources.configMaps"), true)
	conf.FetchSecrets = parseBool(conf.etcConfig("collector.resources.secrets"), false)
	conf.FetchDeployments = parseBool(conf.etcConfig("collector.resources.deployments"), true)
	conf.FetchDaemonSets = parseBool(conf.etcConfig("collector.resources.daemonSets"), true)
	conf.FetchReplicaSets = parseBool(conf.etcConfig("collector.resources.replicaSets"), true)
	conf.FetchStatefulSets = parseBool(conf.etcConfig("collector.resources.statefulSets"), true)
	conf.FetchJobs = parseBool(conf.etcConfig("collector.resources.jobs"), true)
	conf.FetchCronJobs = parseBool(conf.etcConfig("collector.resources.cronJobs"), true)
	conf.FetchIngresses = parseBool(conf.etcConfig("collector.resources.ingresses"), true)
	conf.FetchClusterRoles = parseBool(conf.etcConfig("collector.resources.clusterRoles"), true)

	return conf, nil
}

// IgnoreNamespace accepts a namespace and returns a boolean value indicating
// whether the namespace should be ignored
func (conf *Config) IgnoreNamespace(ns string) bool {
	if conf.Namespace != "" && ns != conf.Namespace {
		return false
	}

	if len(conf.IgnoreNamespaces) > 0 {
		return includes(conf.IgnoreNamespaces, ns)
	}

	return false
}

func parseOne(str, defVal string) string {
	str = strings.TrimSpace(str)
	if str == "" {
		return defVal
	}

	return str
}

func parseMultiple(str string, defVal []string) []string {
	str = strings.TrimSpace(str)
	if str == "" {
		return defVal
	}

	return strings.Split(str, "\n")
}

func parseBool(str string, defVal bool) bool {
	str = strings.TrimSpace(str)
	if str == "" {
		return defVal
	}

	asBool, err := strconv.ParseBool(str)
	if err != nil {
		return defVal
	}

	return asBool
}

func includes(list []string, value string) bool {
	for _, val := range list {
		if val == value {
			return true
		}
	}

	return false
}

func (conf *Config) etcConfig(name string) string {
	data, err := fs.ReadFile(
		conf.FS,
		fmt.Sprintf("%s/%s", strings.TrimPrefix(conf.ConfigDir, "/"), name),
	)
	if err != nil {
		// only log this error if it's _not_ a "no such file or directory"
		// error
		if !os.IsNotExist(err) {
			conf.Log.Warn().
				Err(err).
				Str("key", name).
				Msg("Failed loading configuration key")
		}
		return ""
	}

	return string(data)
}

type localFS struct{}

func (fs *localFS) Open(name string) (fs.File, error) {
	return os.Open("/" + name)
}
