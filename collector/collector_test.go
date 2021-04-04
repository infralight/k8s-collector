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
						Kind     string `json:"kind"`
						Metadata struct {
							Name      string `json:"name"`
							Namespace string `json:"namespace"`
						} `json:"metadata"`
					}
				}
				err := json.NewDecoder(reader).Decode(&data)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// convert the objects to a list of strings
				objs := make([]string, len(data.Objects))
				for i, obj := range data.Objects {
					objs[i] = fmt.Sprintf("%s:%s:%s", obj.Kind, obj.Metadata.Namespace, obj.Metadata.Name)
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

			// Add a ConfigMap that points the collector to talk to the mock
			// HTTP server
			test.objs = append(test.objs, &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind: "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      DefaultConfigMapName,
					Namespace: "default",
				},
				Data: map[string]string{
					"endpoint":                       ts.URL,
					"collector.resources.configMaps": "false",
				},
			})

			// Create a fake Kubernetes client
			client := fake.NewSimpleClientset(test.objs...)

			// create a collector instance
			f := NewCollector(&logger, client)

			ctx := context.Background()

			os.Setenv(AccessKeyEnvVar, "bla")
			os.Setenv(SecretKeyEnvVar, "bla")
			err := f.loadConfig(ctx)
			if err != nil {
				t.Fatalf("Failed configuring collector: %s", err)
			}

			// run the collector
			objects := f.collect(ctx)

			err = f.send(objects)
			if test.expErr {
				assert.MustNotBeNil(t, err, "error must not be nil")
			} else {
				assert.MustBeNil(t, err, "error must be nil")
			}
		})
	}
}
