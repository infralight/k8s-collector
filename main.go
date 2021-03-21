package main

import (
	"context"
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/infralight/k8s-fetcher/fetcher"
)

func main() {
	// Enable a debug flag for more verbose logging
	debug := flag.Bool("debug", false, "sets log level to debug")
	external := flag.String("external", "", "run outside of the cluster (provide path to kubeconfig file)")
	flag.Parse()

	// When running in debug mode, enable pretty-printed logging with minimum
	// log level set at DEBUG. In non-debug mode, use standard JSON logging with
	// unix timestamp for better performance
	if *debug {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}

	var config *rest.Config
	var err error
	if *external != "" {
		config, err = clientcmd.BuildConfigFromFlags("", *external)
	} else {
		// Load configuration to connect to the Kubernetes API from within a K8s
		// cluster
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Panic().
			Err(err).
			Msg("Failed loading Kubernetes configuration")
	}

	// Create a new instance of the Kubernetes API client
	api, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panic().
			Err(err).
			Msg("Failed getting K8s client set")
	}

	f := fetcher.NewFetcher(&log.Logger, api)

	err = f.Run(context.TODO(), os.Stdout)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Fetcher failed")
	}

	log.Info().Msg("Fetched successfully finished")
}
