package config

import (
	"errors"
	"os"
	"testing"
	"testing/fstest"

	"github.com/jgroeneveld/trial/assert"
	"github.com/rs/zerolog"
)

func TestLoadConfig(t *testing.T) {
	// Create a nop logger
	logger := zerolog.Nop()

	var tests = []struct {
		name      string
		moreInfo  []string
		accessKey string
		secretKey string
		etcFiles  *fstest.MapFS
		expErr    error
		expConfig Config
	}{
		{
			name:   "When authentication keypair is missing, loadConfig should fail",
			expErr: ErrAccessKeys,
		},
		{
			name: "When configuration is provided, it should be parsed successfully",
			moreInfo: []string{
				"If an API endpoint is provided, a trailing slash should be trimmed",
			},
			accessKey: "access",
			secretKey: "secret",
			etcFiles: &fstest.MapFS{
				"etc/config/endpoint": &fstest.MapFile{
					Data: []byte("http://localhost:5000/\n"),
				},
				"etc/config/collector.watchNamespace": &fstest.MapFile{
					Data: []byte("namespace"),
				},
				"etc/config/collector.ignoreNamespaces": &fstest.MapFile{
					Data: []byte("one\ntwo\n\n"),
				},
				"etc/config/collector.resources": &fstest.MapFile{
					Data: []byte(
						"\nconfigmaps\nreplicationcontrollers\nsecrets\nservices\nserviceaccounts\npods\nnodes\napplications\n",
					),
				},
			},
			expConfig: Config{
				Log:              &logger,
				ConfigDir:        DefaultConfigDir,
				AccessKey:        "access",
				SecretKey:        "secret",
				Endpoint:         "http://localhost:5000",
				LoginEndpoint:    DefaultFireflyLoginAPI,
				Namespace:        "namespace",
				IgnoreNamespaces: []string{"one", "two"},
				AllowedResources: map[string]bool{
					"configmaps":             true,
					"replicationcontrollers": true,
					"secrets":                true,
					"services":               true,
					"serviceaccounts":        true,
					"pods":                   true,
					"nodes":                  true,
					"applications":           true,
				},
				OverrideUniqueClusterId: false,
				PageSize:                500,
				MaxGoRoutines:           50,
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
			conf, err := LoadConfig(&logger, memFs, "", false)
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
