package filter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/infralight/k8s-collector/collector/k8s"
	"github.com/rs/zerolog/log"
	"github.com/thoas/go-funk"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	argotime "helm.sh/helm/v3/pkg/time"
)

func ArgoFilter(ctx context.Context, data map[string][]interface{}) error {
	for _, value := range data["k8s_objects"] {
		obj, ok := value.(k8s.KubernetesObject)
		if !ok {
			continue
		}

		if obj.Kind != "Application" {
			continue
		}

		meta, ok := obj.Object.(map[string]interface{})
		if !ok {
			continue
		}

		source, _ := funk.Get(meta, "status.sourceType").(string)
		if source != "Helm" {
			continue
		}

		name, _ := funk.Get(meta, "metadata.name").(string)
		namespace, _ := funk.Get(meta, "metadata.namespace").(string)

		r := &release.Release{
			Name:      name,
			Namespace: namespace,
			Info: &release.Info{
				Status: convertK8sStatusToArgoStatus(funk.Get(meta, "status.health.status")),
			},
		}

		if history, ok := funk.Get(meta, "status.history").([]interface{}); ok {
			r.Version, _ = funk.Get(history[len(history)-1], "id").(int)
			for i := 0; i < len(history); i++ {
				if funk.Contains(history[i], "deployedAt") {
					if deployedAt, ok := funk.Get(history[i], "deployedAt").(string); ok {
						dt, _ := argotime.Parse(time.RFC3339, deployedAt)
						if r.Info.FirstDeployed.IsZero() {
							r.Info.FirstDeployed = dt
						}
						r.Info.LastDeployed = dt
					}
				}
			}
		}

		home, _ := funk.Get(meta, "spec.source.repoURL").(string)
		if strings.HasPrefix(home, "https://github.com") && strings.HasSuffix(home, ".git") {
			home = strings.TrimSuffix(home, ".git")
		}

		if strings.HasSuffix(home, "/") {
			home = strings.TrimSuffix(home, "/")
		}

		chartVersion, _ := funk.Get(meta, "spec.source.targetRevision").(string)

		r.Chart = &chart.Chart{
			Metadata: &chart.Metadata{
				Name:       name,
				Type:       "application",
				Home:       home,
				Version:    chartVersion,
				APIVersion: "v2",
			},
		}

		if resources, ok := funk.Get(meta, "status.resources").([]interface{}); ok {
			var yaml strings.Builder
			fmt.Fprintln(&yaml, "---")
			for i, ires := range resources {
				if res, ok := ires.(map[string]interface{}); ok {
					resApiVersion, _ := funk.Get(res, "version").(string)
					resGroup, _ := funk.Get(res, "group").(string)
					if resGroup != "" {
						resApiVersion = fmt.Sprintf("%s/%s", resGroup, resApiVersion)
					}
					resKind, _ := funk.Get(res, "kind").(string)
					resName, _ := funk.Get(res, "name").(string)
					resNamespace, _ := funk.Get(res, "namespace").(string)
					fmt.Fprintf(&yaml, "apiVersion: %s\n", resApiVersion)
					fmt.Fprintf(&yaml, "kind: %s\n", resKind)
					fmt.Fprintf(&yaml, "metadata:\n")
					fmt.Fprintf(&yaml, "  name: %s\n", resName)
					if resNamespace != "" {
						fmt.Fprintf(&yaml, "  namespace: %s\n", resNamespace)
					}
					fmt.Fprintf(&yaml, "  labels:\n")
					fmt.Fprintf(&yaml, "    helm.sh/chart: %s\n", name)
					fmt.Fprintf(&yaml, "    argocd.argoproj.io/instance: %s\n", name)
					if i < len(resources)-1 {
						fmt.Fprintf(&yaml, "---\n")
					}
				}
			}
			r.Manifest = yaml.String()
		}

		data["helm_releases"] = append(data["helm_releases"], r)

		log.Info().Str("name", name).Msg("Found Helm chart in Argo app")
	}

	return nil
}

// # Source: file path
// apiVersion: v1
// kind: ServiceAccount
// metadata:
//   name: infralight-service-account
// ---
// # Source: file path
// apiVersion: v1
// kind: Secret
// metadata:
//   name: infralight-credentials
//   namespace: default
//   type: Opaque
//   data: ...

// https://github.com/argoproj/gitops-engine/blob/master/pkg/health/health.go
func convertK8sStatusToArgoStatus(val interface{}) release.Status {
	if val == nil {
		return release.StatusUnknown
	}

	str, ok := val.(string)
	if !ok {
		return release.StatusUnknown
	}

	switch strings.ToLower(str) {
	case "degraded", "missing":
		return release.StatusFailed
	case "unknown":
		return release.StatusUnknown
	}

	return release.StatusDeployed
}
