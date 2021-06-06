package collector

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"testing"
	"testing/fstest"

	"github.com/infralight/k8s-collector/collector/config"
	"github.com/infralight/k8s-collector/collector/k8s"
	"github.com/jgroeneveld/trial/assert"
	"github.com/rs/zerolog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRun(t *testing.T) {
	// Create a nop logger
	logger := zerolog.Nop()

	var tests = []struct {
		name       string
		objs       []runtime.Object
		expObjects []string
		expErr     bool
	}{
		{
			name: "simple test",
			objs: []runtime.Object{
				&v1.Pod{
					TypeMeta: metav1.TypeMeta{Kind: "Pod"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-pod",
						Namespace: "default",
					},
				},
				&v1.Namespace{
					TypeMeta: metav1.TypeMeta{Kind: "Namespace"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default",
						Namespace: "default",
					},
				},
				&v1.ConfigMap{
					TypeMeta: metav1.TypeMeta{Kind: "ConfigMap"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-configmap",
						Namespace: "default",
					},
				},
			},
			expObjects: []string{
				"Pod:default:fake-pod",
				"Namespace:default:default",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Start a mock HTTP server that will accept the fake data
			// we generate, and only return 204 No Content if the data matches
			// the test's expectations
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost && r.URL.Path == "/account/access_keys/login" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"access_token":"token"}`))
					return
				}

				// read the data we've received and decode it into an array
				// of Kubernetes objects
				reader := r.Body
				if r.Header.Get("Content-Encoding") == "gzip" {
					var err error
					reader, err = gzip.NewReader(r.Body)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					defer reader.Close()
				}

				var data struct {
					Objects []struct {
						Kind   string `json:"kind"`
						Object struct {
							Metadata struct {
								Name      string `json:"name"`
								Namespace string `json:"namespace"`
							} `json:"metadata"`
						} `json:"object"`
					} `json:"k8s_objects"`
				}
				err := json.NewDecoder(reader).Decode(&data)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// convert the objects to a list of strings
				objs := make([]string, len(data.Objects))
				for i, obj := range data.Objects {
					if obj.Kind == "ConfigMap" {
						// Should be filtered by Conf
						t.Error("ConfigMap should be filtered")
					}
					objs[i] = fmt.Sprintf("%s:%s:%s", obj.Kind, obj.Object.Metadata.Namespace, obj.Object.Metadata.Name)
				}

				sort.Strings(objs)
				sort.Strings(test.expObjects)
				if reflect.DeepEqual(objs, test.expObjects) {
					w.WriteHeader(http.StatusNoContent)
					return
				}

				t.Logf("Expected %v, got %v", test.expObjects, objs)
				w.WriteHeader(http.StatusBadRequest)
			}))

			defer ts.Close()

			// Load configuration
			os.Setenv(config.AccessKeyEnvVar, "bla")
			os.Setenv(config.SecretKeyEnvVar, "bla")
			conf, err := config.LoadConfig(&logger, &fstest.MapFS{
				"etc/config/endpoint":                       &fstest.MapFile{Data: []byte(ts.URL)},
				"etc/config/collector.resources.configMaps": &fstest.MapFile{Data: []byte("false")},
			}, "")
			if err != nil {
				t.Fatalf("Unexpectedly failed loading configuration: %s", err)
			}

			// Load the Kubernetes collector with a fake K8s client
			k8sCollector := k8s.New(fake.NewSimpleClientset(test.objs...))

			// create and run the collector
			err = New("test", conf, k8sCollector).Run(context.Background())
			if test.expErr {
				assert.MustNotBeNil(t, err, "error must not be nil")
			} else {
				assert.MustBeNil(t, err, "error must be nil")
			}
		})
	}
}
