package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/DataDog/zstd"
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
		name        string
		validAPIKey string
		testAPIKey  string
		objs        []runtime.Object
		expObjects  []string
		expErr      bool
	}{
		{
			name:        "invalid API key",
			validAPIKey: "key",
			testAPIKey:  "kei",
			expErr:      true,
		},
		{
			name:        "simple test",
			validAPIKey: "key",
			testAPIKey:  "key",
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
				if r.Header.Get("Authorization") != fmt.Sprintf("Bearer %s", test.validAPIKey) {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				// read the data we've received and decode it into an array
				// of Kubernetes objects
				body, err := io.ReadAll(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				if r.Header.Get("Content-Encoding") == "zstd" {
					body, err = zstd.Decompress(nil, body)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
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
				err = json.Unmarshal(body, &data)
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

			// Add a ConfigMap that points the fetcher to talk to the mock
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
					"endpoint":                     ts.URL,
					"fetcher.resources.configMaps": "false",
				},
			})

			// Create a fake Kubernetes client
			client := fake.NewSimpleClientset(test.objs...)

			// create a fetcher instance
			os.Setenv(APIKeyEnvVar, test.testAPIKey)
			f := NewFetcher(&logger, client)

			// run the fetcher
			err := f.Run(context.Background())
			if test.expErr {
				assert.MustNotBeNil(t, err, "error must not be nil")
			} else {
				assert.MustBeNil(t, err, "error must be nil")
			}
		})
	}
}
