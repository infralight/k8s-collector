package main

import (
	"context"
	"flag"
	"os"

	"github.com/infralight/k8s-collector/collector"
	"github.com/infralight/k8s-collector/collector/config"
	"github.com/infralight/k8s-collector/collector/helm"
	"github.com/infralight/k8s-collector/collector/k8s"
	"github.com/infralight/k8s-collector/collector/k8stypes"
	Logger "github.com/infralight/logger"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	// Parse command line flags
	debug := flag.Bool("debug", false, "sets log level to debug")
	external := flag.String("external", "", "run outside of the cluster (provide path to kubeconfig file)")
	configDir := flag.String("config", "/etc/config", "configuration files directory")
	flag.Parse()
	if *debug {
		_ = os.Setenv("INFRALIGHT_ENV", "development")
	}

	// Initiate a logger
	logger := Logger.New()
	defer logger.Flush()

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

	// Load the Kubernetes collector
	k8sCollector, err := k8s.DefaultConfiguration(*external)
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Failed loading Kubernetes collector")
	}

	k8sTypesCollector, err := k8stypes.DefaultConfiguration(*external)
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
		New(clusterID, conf, k8sCollector, helmCollector, k8sTypesCollector).
		Run(context.TODO())
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("Fetcher failed")
	}

	logger.Info().Msg("Fetcher successfully finished")
}
