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

	// DefaultFireflyAPI is the default URL for Firefly's API
	DefaultFireflyAPI = "https://k8s-api.prod.external.api.infralight.cloud"

	// DefaultFireflyLoginAPI is the default URL for Firefly's Login API
	DefaultFireflyLoginAPI = "https://prod.external.api.infralight.cloud"
)

var (
	// ErrAccessKeys is an error returned when the environment variables for the
	// access and secret keys are not provided or empty.
	ErrAccessKeys = errors.New("access and secret keys must be provided")

	// ErrEndpoint is an error returned when the configuration directory is
	// missing an endpoint setting (endpoint is the URL to the Infralight App
	// Server).
	ErrEndpoint = errors.New("Infralight endpoint must be provided")

	// DefaultResourceTypes is the list of Kubernetes resources that are
	// to be collected by default (i.e. if there is no configuration at all)
	DefaultResourceTypes = []string{
		"apiservices",
		"analysistemplates",
		"clusteranalysistemplates",
		"clusterroles",
		"clusterrolebindings",
		"configmaps",
		"controllerrevisions",
		"csinodes",
		"cronjobs",
		"customresourcedefinitions",
		"daemonsets",
		"deployments",
		"endpoints",
		"endpointslices",
		"flowschemas",
		"ingresses",
		"jobs",
		"leases",
		"namespaces",
		"networkpolicies",
		"nodes",
		"persistentvolumeclaims",
		"persistentvolumes",
		"pods",
		"priorityclasses",
		"prioritylevelconfigurations",
		"replicasets",
		"replicationcontrollers",
		"roles",
		"rolebindings",
		"rollouts",
		"rollouts/finalizers",
		"rollouts/status",
		"serviceaccounts",
		"services",
		"services/status",
		"statefulsets",
		"storageclasses",
	}
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

	// DryRun indicates whether the collector should only perform local read
	// operations. When true, authentication against the Firefly API is not
	// made, as is sending of collected data. Data is printed to standard output
	// instead
	DryRun bool

	// The logger instance
	Log *zerolog.Logger

	// AccessKey is the Infralight access key
	AccessKey string

	// SecretKey is the Infralight secret key
	SecretKey string

	// Endpoint is the URL to the Infralight App Server
	Endpoint string

    // LoginEndpoint is the URL to login Infralight Service
    LoginEndpoint string

	// Namespace is the Kubernets namespace we're collecting data from (if empty,
	// all namespaces are collected)
	Namespace string

	// IgnoreNamespaces is a list of namespaces to ignore (only taken into
	// account when Namespace is empty)
	IgnoreNamespaces []string

	// AllowedResources is a list of resource types (named by their "Kind" value)
	// that the collector is allowed to collect
	AllowedResources map[string]bool

	// OverrideUniqueClusterId is a boolean indicating whether to override the master url of the Kubernetes integration
	OverrideUniqueClusterId bool

	// PageSize is an integer for max page size in KB
	PageSize int

	// MaxGoRoutines is an integer for max goroutines running at ones sending the chunks.
	MaxGoRoutines int
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
	dryRun bool,
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
	if !dryRun && (accessKey == "" || secretKey == "") {
		return conf, ErrAccessKeys
	}

	conf = &Config{
		FS:        cfs,
		ConfigDir: configDir,
		Log:       log,
		DryRun:    dryRun,
	}

	conf.Endpoint = strings.TrimSuffix(
		parseOne(conf.etcConfig("endpoint"), ""),
		"/",
	)
	if conf.Endpoint == "" || isOldEndpoint(conf.Endpoint) {
		conf.Endpoint = DefaultFireflyAPI
	}

    conf.LoginEndpoint = strings.TrimSuffix(
        parseOne(conf.etcConfig("loginEndpoint"), ""),
        "/",
    )
    if conf.LoginEndpoint == "" {
        conf.LoginEndpoint = DefaultFireflyLoginAPI
    }

	conf.AccessKey = accessKey
	conf.SecretKey = secretKey
	conf.Namespace = parseOne(conf.etcConfig("collector.watchNamespace"), "")
	conf.IgnoreNamespaces = parseMultiple(conf.etcConfig("collector.ignoreNamespaces"), nil)

	conf.AllowedResources = make(map[string]bool)
	conf.backwardsCompatibilityResources()
	for _, resource := range parseMultiple(conf.etcConfig("collector.resources"), DefaultResourceTypes) {
		conf.AllowedResources[resource] = true
	}

	conf.OverrideUniqueClusterId = parseBool(
		conf.etcConfig("collector.OverrideUniqueClusterId"),
		false,
	)
	conf.PageSize = parseInt(conf.etcConfig("collector.PageSize"), 500)
	conf.MaxGoRoutines = parseInt(conf.etcConfig("collector.MaxGoRoutines"), 50)

	return conf, nil
}

func (conf *Config) backwardsCompatibilityResources() {
	entries, err := fs.ReadDir(conf.FS, conf.ConfigDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "collector.resources.") {
			name := strings.ToLower(strings.TrimPrefix(entry.Name(), "collector.resources."))
			conf.AllowedResources[name] = parseBool(conf.etcConfig(entry.Name()), false)
		}
	}
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

func parseInt(str string, defVal int) int {
	str = strings.TrimSpace(str)
	asInt, err := strconv.Atoi(str)
	if err != nil {
		return defVal
	}
	return asInt
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

func isOldEndpoint(endpoint string) bool {
	return endpoint == "https://prodapi.infralight.cloud/api"
}
