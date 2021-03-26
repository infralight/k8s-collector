package collector

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Collector) getNamespaces(ctx context.Context) (
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

func (f *Collector) getPods(ctx context.Context) (
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

func (f *Collector) getEvents(ctx context.Context) (
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

func (f *Collector) getConfigMaps(ctx context.Context) (
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

func (f *Collector) getReplicationControllers(ctx context.Context) (
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

func (f *Collector) getNodes(ctx context.Context) (
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

func (f *Collector) getServices(ctx context.Context) (
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

func (f *Collector) getServiceAccounts(ctx context.Context) (
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

func (f *Collector) getPersistentVolumes(ctx context.Context) (
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

func (f *Collector) getPersistentVolumeClaims(ctx context.Context) (
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

func (f *Collector) getSecrets(ctx context.Context) (
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

func (f *Collector) getDeployments(ctx context.Context) (
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

func (f *Collector) getDaemonSets(ctx context.Context) (
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

func (f *Collector) getReplicaSets(ctx context.Context) (
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

func (f *Collector) getStatefulSet(ctx context.Context) (
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

func (f *Collector) getJobs(ctx context.Context) (
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

func (f *Collector) getCronJobs(ctx context.Context) (
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

func (f *Collector) getIngresses(ctx context.Context) (
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

func (f *Collector) getClusterRoles(ctx context.Context) (
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
