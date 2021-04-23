package collector

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

const (
	AccessKeyEnvVar = "INFRALIGHT_ACCESS_KEY"
	SecretKeyEnvVar = "INFRALIGHT_SECRET_KEY" // nolint: gosec
)

var (
	ErrAccessKeys = errors.New("access and secret keys must be provided")
	ErrEndpoint   = errors.New("Infralight endpoint must be provided")
)

type CollectorConfig struct {
	AccessKey                   string
	SecretKey                   string
	Endpoint                    string
	Namespace                   string
	IgnoreNamespaces            []string
	FetchEvents                 bool
	FetchConfigMaps             bool
	FetchReplicationControllers bool
	FetchSecrets                bool
	FetchServices               bool
	FetchServiceAccounts        bool
	FetchPods                   bool
	FetchNodes                  bool
	FetchPersistentVolumes      bool
	FetchPersistentVolumeClaims bool
	FetchNamespaces             bool
	FetchDeployments            bool
	FetchDaemonSets             bool
	FetchReplicaSets            bool
	FetchStatefulSets           bool
	FetchJobs                   bool
	FetchCronJobs               bool
	FetchIngresses              bool
	FetchClusterRoles           bool
}

func (f *Collector) loadConfig() error {
	// load Infralight API Key from the environment, this is required
	accessKey := os.Getenv(AccessKeyEnvVar)
	secretKey := os.Getenv(SecretKeyEnvVar)
	if accessKey == "" || secretKey == "" {
		return ErrAccessKeys
	}

	f.config = &CollectorConfig{
		AccessKey:                   accessKey,
		SecretKey:                   secretKey,
		Endpoint:                    strings.TrimSuffix(parseOne(f.etcConfig("endpoint"), ""), "/"),
		Namespace:                   parseOne(f.etcConfig("collector.watchNamespace"), ""),
		IgnoreNamespaces:            parseMultiple(f.etcConfig("collector.ignoreNamespaces"), nil),
		FetchEvents:                 parseBool(f.etcConfig("collector.resources.events"), true),
		FetchReplicationControllers: parseBool(f.etcConfig("collector.resources.replicationControllers"), true),
		FetchServices:               parseBool(f.etcConfig("collector.resources.services"), true),
		FetchServiceAccounts:        parseBool(f.etcConfig("collector.resources.serviceAccounts"), true),
		FetchPods:                   parseBool(f.etcConfig("collector.resources.pods"), true),
		FetchNodes:                  parseBool(f.etcConfig("collector.resources.nodes"), true),
		FetchPersistentVolumes:      parseBool(f.etcConfig("collector.resources.persistentVolumes"), true),
		FetchPersistentVolumeClaims: parseBool(f.etcConfig("collector.resources.persistentVolumeClaims"), true),
		FetchNamespaces:             parseBool(f.etcConfig("collector.resources.namespaces"), true),
		FetchConfigMaps:             parseBool(f.etcConfig("collector.resources.configMaps"), true),
		FetchSecrets:                parseBool(f.etcConfig("collector.resources.secrets"), false),
		FetchDeployments:            parseBool(f.etcConfig("collector.resources.deployments"), true),
		FetchDaemonSets:             parseBool(f.etcConfig("collector.resources.daemonSets"), true),
		FetchReplicaSets:            parseBool(f.etcConfig("collector.resources.replicaSets"), true),
		FetchStatefulSets:           parseBool(f.etcConfig("collector.resources.statefulSets"), true),
		FetchJobs:                   parseBool(f.etcConfig("collector.resources.jobs"), true),
		FetchCronJobs:               parseBool(f.etcConfig("collector.resources.cronJobs"), true),
		FetchIngresses:              parseBool(f.etcConfig("collector.resources.ingresses"), true),
		FetchClusterRoles:           parseBool(f.etcConfig("collector.resources.clusterRoles"), true),
	}

	if f.config.Endpoint == "" {
		return ErrEndpoint
	}

	f.log.Info().Interface("map", f.config).Msg("Loaded collector configuration")

	return nil
}

func (config *CollectorConfig) ignoreNamespace(ns string) bool {
	if config.Namespace != "" && ns != config.Namespace {
		return false
	}

	if len(config.IgnoreNamespaces) > 0 {
		return includes(config.IgnoreNamespaces, ns)
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

func (f *Collector) etcConfig(name string) string {
	data, err := fs.ReadFile(
		f.fs,
		fmt.Sprintf("%s/%s", strings.TrimPrefix(f.configDir, "/"), name),
	)
	if err != nil {
		// only log this error if it's _not_ a "no such file or directory"
		// error
		if !os.IsNotExist(err) {
			f.log.Warn().
				Err(err).
				Str("key", name).
				Msg("Failed loading configuration key")
		}
		return ""
	}

	return string(data)
}
