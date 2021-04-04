package collector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AccessKeyEnvVar      = "INFRALIGHT_ACCESS_KEY"
	SecretKeyEnvVar      = "INFRALIGHT_SECRET_KEY" // nolint: gosec
	DefaultEndpoint      = "https://prodapi.infralight.cloud/sink"
	DefaultNamespace     = "default"
	DefaultConfigMapName = "infralight-k8s-collector-config"
)

var (
	ErrAccessKeys = errors.New("access and secret keys must be provided")
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

func (f *Collector) loadConfig(ctx context.Context) error {
	// load Infralight API Key from the environment, this is required
	accessKey := os.Getenv(AccessKeyEnvVar)
	secretKey := os.Getenv(SecretKeyEnvVar)
	if accessKey == "" || secretKey == "" {
		return ErrAccessKeys
	}

	// now load our optional ConfigMap from Kubernetes
	config, err := f.api.CoreV1().
		ConfigMaps(f.namespace).
		Get(ctx, f.configMapName, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) {
			// configuration doesn't exist, warn but do not fail, we'll use our
			// defaults
			f.log.Warn().
				Str("namespace", f.namespace).
				Str("config_map_name", f.configMapName).
				Msg("ConfigMap doesn't exist, using defaults")

			config = &v1.ConfigMap{
				Data: make(map[string]string),
			}
		} else {
			return fmt.Errorf("failed loading ConfigMap: %w", err)
		}
	}

	f.config = &CollectorConfig{
		AccessKey:                   accessKey,
		SecretKey:                   secretKey,
		Endpoint:                    strings.TrimSuffix(parseOne(config.Data["endpoint"], DefaultEndpoint), "/"),
		Namespace:                   parseOne(config.Data["collector.watchNamespace"], ""),
		IgnoreNamespaces:            parseMultiple(config.Data["collector.ignoreNamespaces"], nil),
		FetchEvents:                 parseBool(config.Data["collector.resources.events"], true),
		FetchReplicationControllers: parseBool(config.Data["collector.resources.replicationControllers"], true),
		FetchServices:               parseBool(config.Data["collector.resources.services"], true),
		FetchServiceAccounts:        parseBool(config.Data["collector.resources.serviceAccounts"], true),
		FetchPods:                   parseBool(config.Data["collector.resources.pods"], true),
		FetchNodes:                  parseBool(config.Data["collector.resources.nodes"], true),
		FetchPersistentVolumes:      parseBool(config.Data["collector.resources.persistentVolumes"], true),
		FetchPersistentVolumeClaims: parseBool(config.Data["collector.resources.persistentVolumeClaims"], true),
		FetchNamespaces:             parseBool(config.Data["collector.resources.namespaces"], true),
		FetchConfigMaps:             parseBool(config.Data["collector.resources.configMaps"], true),
		FetchSecrets:                parseBool(config.Data["collector.resources.secrets"], false),
		FetchDeployments:            parseBool(config.Data["collector.resources.deployments"], true),
		FetchDaemonSets:             parseBool(config.Data["collector.resources.daemonSets"], true),
		FetchReplicaSets:            parseBool(config.Data["collector.resources.replicaSets"], true),
		FetchStatefulSets:           parseBool(config.Data["collector.resources.statefulSets"], true),
		FetchJobs:                   parseBool(config.Data["collector.resources.jobs"], true),
		FetchCronJobs:               parseBool(config.Data["collector.resources.cronJobs"], true),
		FetchIngresses:              parseBool(config.Data["collector.resources.ingresses"], true),
		FetchClusterRoles:           parseBool(config.Data["collector.resources.clusterRoles"], true),
	}

	f.log.Debug().Interface("map", f.config).Msg("Loaded config map")

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
