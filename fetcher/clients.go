package fetcher

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Fetcher) getNamespaces(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getPods(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getEvents(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getConfigMaps(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getReplicationControllers(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.CoreV1().ReplicationControllers("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getNodes(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getServices(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getServiceAccounts(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.CoreV1().ServiceAccounts("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getPersistentVolumes(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getPersistentVolumeClaims(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getSecrets(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getDeployments(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getDaemonSets(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getReplicaSets(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getStatefulSet(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getJobs(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.BatchV1().Jobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getCronJobs(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.BatchV1beta1().CronJobs("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getIngresses(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (f *Fetcher) getClusterRoles(ctx context.Context) (
	items []interface{},
	err error,
) {
	list, err := f.api.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return items, err
	}

	for _, item := range list.Items {
		if f.config.ignoreNamespace(item.Namespace) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}
