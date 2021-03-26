package collector

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jgroeneveld/trial/assert"
	"github.com/rs/zerolog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_loadConfig(t *testing.T) {
	// Create a nop logger
	logger := zerolog.Nop()

	var tests = []struct {
		name      string
		apiKey    string
		objs      []runtime.Object
		expErr    error
		expConfig CollectorConfig
	}{
		{
			name:   "no secret, no config",
			expErr: ErrAPIKey,
		},
		{
			name:   "secret, no config",
			apiKey: "bla",
			expConfig: CollectorConfig{
				APIKey:                      "bla",
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
			name:   "secret, config",
			apiKey: "bla",
			objs: []runtime.Object{
				&v1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind: "ConfigMap",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      DefaultConfigMapName,
						Namespace: "default",
					},
					Data: map[string]string{
						"endpoint":                        "http://localhost:5000/\n",
						"collector.watchNamespace":        "namespace",
						"collector.ignoreNamespaces":      "one\ntwo\n\n",
						"collector.resources.secrets":     "\ntrue   \n",
						"collector.resources.deployments": "false\n",
					},
				},
			},
			expConfig: CollectorConfig{
				APIKey:                      "bla",
				Endpoint:                    "http://localhost:5000/",
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
			client := fake.NewSimpleClientset(test.objs...)

			// create a collector instance
			f := NewCollector(&logger, client)

			if test.apiKey != "" {
				os.Setenv(APIKeyEnvVar, test.apiKey)
			} else {
				os.Unsetenv(APIKeyEnvVar)
			}

			err := f.loadConfig(context.Background())
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
