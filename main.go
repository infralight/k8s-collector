package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/infralight/k8s-collector/collector"
	"github.com/infralight/k8s-collector/collector/config"
	"github.com/infralight/k8s-collector/collector/helm"
	"github.com/infralight/k8s-collector/collector/k8s"
	"github.com/infralight/k8s-collector/collector/k8stypes"
)

func main() {
	// Parse command line flags
	debug := flag.Bool("debug", false, "sets log level to debug")
	external := flag.String("external", "", "run outside of the cluster (provide path to kubeconfig file)")
	configDir := flag.String("config", "/etc/config", "configuration files directory")
	flag.Parse()

	// Initiate a logger
	logger := loadLogger(*debug)

	// Get cluster ID from command line arguments or environment variable.
	// The cluster ID is required.
	clusterID := flag.Arg(0)
	if clusterID == "" {
		clusterID = os.Getenv("CLUSTER_ID")
	}
	if clusterID == "" {
		logger.Fatal().
			Msg("Cluster ID must be provided either as a command line argument, or via the CLUSTER_ID environment variable")
	}

	// Load the collector configuration
	conf, err := config.LoadConfig(logger, nil, *configDir)
	if err != nil {
		logger.Panic().
			Err(err).
			Msg("Failed loading collector configuration")
	}

	apiConfig, err := loadKubeConfig(*external)
	if err != nil {
		logger.Panic().
			Err(err).
			Msg("Failed loading Kubernetes configuration")
	}

	// Load the Kubernetes collector
	k8sCollector, err := k8s.DefaultConfiguration(apiConfig)
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed loading Kubernetes collector")
	}

	k8sTypesCollector, err := k8stypes.DefaultConfiguration(apiConfig)
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed loading Kubernetes collector")
	}

	// Load the Helm collector
	helmCollector, err := helm.DefaultConfiguration(logger.Printf)
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed loading Helm collector")
	}

	err = collector.
		New(clusterID, apiConfig, conf, k8sCollector, helmCollector, k8sTypesCollector).
		Run(context.TODO())
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Fetcher failed")
	}

	logger.Info().Msg("Fetcher successfully finished")
}

func loadKubeConfig(external string) (apiConfig *rest.Config, err error) {
	// Load configuration for the Kubernetes API client. We are either running
	// from inside the cluster (i.e. inside a pod) or outside of the cluster.
	if external != "" {
		apiConfig, err = clientcmd.BuildConfigFromFlags("", external)
	} else {
		// Load configuration to connect to the Kubernetes API from within a K8s
		// cluster
		apiConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return apiConfig, fmt.Errorf("failed loading Kubernetes configuration: %w", err)
	}
	return apiConfig, nil
}

func loadLogger(debug bool) *zerolog.Logger {
	// When running in debug mode, enable pretty-printed logging with minimum
	// log level set at DEBUG. In non-debug mode, use standard JSON logging with
	// unix timestamp for better performance
	if debug {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}

	return &log.Logger
}
