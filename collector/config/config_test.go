package config

import (
	"errors"
	"os"
	"testing"
	"testing/fstest"

	"github.com/jgroeneveld/trial/assert"
	"github.com/rs/zerolog"
)

func Test_loadConfig(t *testing.T) {
	// Create a nop logger
	logger := zerolog.Nop()

	var tests = []struct {
		name      string
		accessKey string
		secretKey string
		etcFiles  *fstest.MapFS
		expErr    error
		expConfig Config
	}{
		{
			name:   "no secret",
			expErr: ErrAccessKeys,
		},
		{
			name:      "no endpoint",
			accessKey: "access",
			secretKey: "secret",
			expErr:    ErrEndpoint,
		},
		{
			name:      "no config",
			accessKey: "access",
			secretKey: "secret",
			etcFiles: &fstest.MapFS{
				"etc/config/endpoint": &fstest.MapFile{Data: []byte("http://localhost:5000/api\n")},
			},
			expConfig: Config{
				Log:                         &logger,
				ConfigDir:                   DefaultConfigDir,
				AccessKey:                   "access",
				SecretKey:                   "secret",
				Endpoint:                    "http://localhost:5000/api",
				FetchEvents:                 true,
				FetchConfigMaps:             true,
				FetchReplicationControllers: true,
				FetchSecrets:                false,
				FetchServices:               true,
				FetchServiceAccounts:        true,
				FetchPods:                   true,
				FetchNodes:                  true,
				FetchPersistentVolumes:      true,
				FetchPersistentVolumeClaims: true,
				FetchNamespaces:             true,
				FetchDeployments:            true,
				FetchDaemonSets:             true,
				FetchReplicaSets:            true,
				FetchStatefulSets:           true,
				FetchJobs:                   true,
				FetchCronJobs:               true,
				FetchIngresses:              true,
				FetchClusterRoles:           true,
			},
		},
		{
			name:      "config",
			accessKey: "access",
			secretKey: "secret",
			etcFiles: &fstest.MapFS{
				"etc/config/endpoint":                        &fstest.MapFile{Data: []byte("http://localhost:5000/api\n")},
				"etc/config/collector.watchNamespace":        &fstest.MapFile{Data: []byte("namespace")},
				"etc/config/collector.ignoreNamespaces":      &fstest.MapFile{Data: []byte("one\ntwo\n\n")},
				"etc/config/collector.resources.secrets":     &fstest.MapFile{Data: []byte("\ntrue   \n")},
				"etc/config/collector.resources.deployments": &fstest.MapFile{Data: []byte("false\n")},
			},
			expConfig: Config{
				Log:                         &logger,
				ConfigDir:                   DefaultConfigDir,
				AccessKey:                   "access",
				SecretKey:                   "secret",
				Endpoint:                    "http://localhost:5000/api",
				Namespace:                   "namespace",
				IgnoreNamespaces:            []string{"one", "two"},
				FetchEvents:                 true,
				FetchConfigMaps:             true,
				FetchReplicationControllers: true,
				FetchSecrets:                true,
				FetchServices:               true,
				FetchServiceAccounts:        true,
				FetchPods:                   true,
				FetchNodes:                  true,
				FetchPersistentVolumes:      true,
				FetchPersistentVolumeClaims: true,
				FetchNamespaces:             true,
				FetchDeployments:            false,
				FetchDaemonSets:             true,
				FetchReplicaSets:            true,
				FetchStatefulSets:           true,
				FetchJobs:                   true,
				FetchCronJobs:               true,
				FetchIngresses:              true,
				FetchClusterRoles:           true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create an in-memory filesystem for configuration files
			memFs := test.etcFiles
			if memFs == nil {
				memFs = &fstest.MapFS{}
			}

			// Set environment variables
			if test.accessKey != "" {
				os.Setenv(AccessKeyEnvVar, test.accessKey)
				os.Setenv(SecretKeyEnvVar, test.secretKey)
			} else {
				os.Unsetenv(AccessKeyEnvVar)
				os.Unsetenv(SecretKeyEnvVar)
			}

			// Load collector configuration
			conf, err := LoadConfig(&logger, memFs, "")
			if test.expErr != nil {
				assert.MustNotBeNil(t, err, "error must not be nil")
				assert.True(t, errors.Is(err, test.expErr), "error must match")
			} else {
				assert.MustBeNil(t, err, "error must be nil")
				conf.FS = nil
				assert.DeepEqual(t, test.expConfig, *conf, "config must match")
			}
		})
	}
}
