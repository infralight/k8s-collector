package collector

import (
	"errors"
	"os"
	"testing"
	"testing/fstest"

	"github.com/jgroeneveld/trial/assert"
	"github.com/rs/zerolog"
	"k8s.io/client-go/kubernetes/fake"
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
		expConfig CollectorConfig
	}{
		{
			name:   "no secret, no config",
			expErr: ErrAccessKeys,
		},
		{
			name:      "secret, no config",
			accessKey: "bla",
			secretKey: "bla2",
			expConfig: CollectorConfig{
				AccessKey:                   "bla",
				SecretKey:                   "bla2",
				Endpoint:                    DefaultEndpoint,
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
			name:      "secret, config",
			accessKey: "bla",
			secretKey: "bla2",
			etcFiles: &fstest.MapFS{
				"etc/config/endpoint":                        &fstest.MapFile{Data: []byte("http://localhost:5000/\n")},
				"etc/config/collector.watchNamespace":        &fstest.MapFile{Data: []byte("namespace")},
				"etc/config/collector.ignoreNamespaces":      &fstest.MapFile{Data: []byte("one\ntwo\n\n")},
				"etc/config/collector.resources.secrets":     &fstest.MapFile{Data: []byte("\ntrue   \n")},
				"etc/config/collector.resources.deployments": &fstest.MapFile{Data: []byte("false\n")},
			},
			expConfig: CollectorConfig{
				AccessKey:                   "bla",
				SecretKey:                   "bla2",
				Endpoint:                    "http://localhost:5000",
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
			// Create a fake Kubernetes client
			client := fake.NewSimpleClientset()

			// Create an in-memory filesystem for configuration files
			memFs := test.etcFiles
			if memFs == nil {
				memFs = &fstest.MapFS{}
			}

			// create a collector instance
			f := NewCollector("test", &logger, client)
			f.SetFS(memFs)

			if test.accessKey != "" {
				os.Setenv(AccessKeyEnvVar, test.accessKey)
				os.Setenv(SecretKeyEnvVar, test.secretKey)
			} else {
				os.Unsetenv(AccessKeyEnvVar)
				os.Unsetenv(SecretKeyEnvVar)
			}

			err := f.loadConfig()
			if test.expErr != nil {
				assert.MustNotBeNil(t, err, "error must not be nil")
				assert.True(t, errors.Is(err, test.expErr), "error must match")
			} else {
				assert.MustBeNil(t, err, "error must be nil")
				assert.DeepEqual(t, test.expConfig, *f.config, "config must match")
			}
		})
	}
}
