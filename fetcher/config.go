package fetcher

import (
	"context"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FetcherConfig struct {
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

func (f *Fetcher) loadConfig(ctx context.Context) error {
	config, err := f.api.CoreV1().
		ConfigMaps(f.namespace).
		Get(ctx, f.configMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	f.config = &FetcherConfig{
		Namespace:                   parseOne(config.Data["fetcher.watchNamespace"], ""),
		IgnoreNamespaces:            parseMultiple(config.Data["fetcher.ignoreNamespaces"], nil),
		FetchEvents:                 parseBool(config.Data["fetcher.resources.events"], true),
		FetchReplicationControllers: parseBool(config.Data["fetcher.resources.replicationControllers"], true),
		FetchServices:               parseBool(config.Data["fetcher.resources.services"], true),
		FetchServiceAccounts:        parseBool(config.Data["fetcher.resources.serviceAccounts"], true),
		FetchPods:                   parseBool(config.Data["fetcher.resources.pods"], true),
		FetchNodes:                  parseBool(config.Data["fetcher.resources.nodes"], true),
		FetchPersistentVolumes:      parseBool(config.Data["fetcher.resources.persistentVolumes"], true),
		FetchPersistentVolumeClaims: parseBool(config.Data["fetcher.resources.persistentVolumeClaims"], true),
		FetchNamespaces:             parseBool(config.Data["fetcher.resources.namespaces"], true),
		FetchConfigMaps:             parseBool(config.Data["fetcher.resources.configMaps"], true),
		FetchSecrets:                parseBool(config.Data["fetcher.resources.secrets"], false),
		FetchDeployments:            parseBool(config.Data["fetcher.resources.deployments"], true),
		FetchDaemonSets:             parseBool(config.Data["fetcher.resources.daemonSets"], true),
		FetchReplicaSets:            parseBool(config.Data["fetcher.resources.replicaSets"], true),
		FetchStatefulSets:           parseBool(config.Data["fetcher.resources.statefulSets"], true),
		FetchJobs:                   parseBool(config.Data["fetcher.resources.jobs"], true),
		FetchCronJobs:               parseBool(config.Data["fetcher.resources.cronJobs"], true),
		FetchIngresses:              parseBool(config.Data["fetcher.resources.ingresses"], true),
		FetchClusterRoles:           parseBool(config.Data["fetcher.resources.clusterRoles"], true),
	}

	f.log.Debug().Interface("map", f.config).Msg("Loaded config map")

	return nil
}

func (config *FetcherConfig) ignoreNamespace(ns string) bool {
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
