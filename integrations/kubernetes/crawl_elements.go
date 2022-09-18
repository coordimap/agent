package kubernetes

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	extensionsBeta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kubeCrawler *kubernetesCrawler) getNodes() ([]v1.Node, error) {
	list, err := kubeCrawler.kubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not load the kubernetes nodes. %w", err)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listNamespaces() ([]v1.Namespace, error) {
	list, err := kubeCrawler.kubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not load the kubernetes namespaces. %w", err)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listPods(namespace string) ([]v1.Pod, error) {
	podList, errPods := kubeCrawler.kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if errPods != nil {
		return nil, fmt.Errorf("could not list the pods for namespace: %s. %w", namespace, errPods)
	}

	return podList.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listDeplyments(namespace string) ([]appsv1.Deployment, error) {
	list, errPods := kubeCrawler.kubeClient.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if errPods != nil {
		return nil, fmt.Errorf("could not list the deployments for namespace: %s. %w", namespace, errPods)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listServices(namespace string) ([]v1.Service, error) {
	list, errPods := kubeCrawler.kubeClient.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if errPods != nil {
		return nil, fmt.Errorf("could not list the services for namespace: %s. %w", namespace, errPods)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listSecrets(namespace string) ([]v1.Secret, error) {
	list, errList := kubeCrawler.kubeClient.CoreV1().Secrets(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the secrets for namespace: %s. %w", namespace, errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listJobs(namespace string) ([]batchv1.Job, error) {
	list, errList := kubeCrawler.kubeClient.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the jobs for namespace: %s. %w", namespace, errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listCronJobs(namespace string) ([]batchv1.CronJob, error) {
	list, errList := kubeCrawler.kubeClient.BatchV1().CronJobs(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the cronjobs for namespace: %s. %w", namespace, errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listEndpoints(namespace string) ([]v1.Endpoints, error) {
	list, errList := kubeCrawler.kubeClient.CoreV1().Endpoints(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the endpoints for namespace: %s. %w", namespace, errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listConfigMaps(namespace string) ([]v1.ConfigMap, error) {
	list, errList := kubeCrawler.kubeClient.CoreV1().ConfigMaps(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the configmaps for namespace: %s. %w", namespace, errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listStatefulSets(namespace string) ([]appsv1.StatefulSet, error) {
	list, errList := kubeCrawler.kubeClient.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the configmaps for namespace: %s. %w", namespace, errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listDaemonSets(namespace string) ([]appsv1.DaemonSet, error) {
	list, errList := kubeCrawler.kubeClient.AppsV1().DaemonSets(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the configmaps for namespace: %s. %w", namespace, errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listStorageClasses() ([]storagev1.StorageClass, error) {
	list, errList := kubeCrawler.kubeClient.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the storageclasses. %w", errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listPersistentVolumeClaims(namespace string) ([]v1.PersistentVolumeClaim, error) {
	list, errList := kubeCrawler.kubeClient.CoreV1().PersistentVolumeClaims(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the storageclasses. %w", errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listPersistentVolumes() ([]v1.PersistentVolume, error) {
	list, errList := kubeCrawler.kubeClient.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the storageclasses. %w", errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listIngressesExtensionsBeta1(namespace string) ([]extensionsBeta1.Ingress, error) {
	list, errList := kubeCrawler.kubeClient.ExtensionsV1beta1().Ingresses(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the ingresses extensions beta1. %w", errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listIngressesNetworkingV1(namespace string) ([]networkingv1.Ingress, error) {
	list, errList := kubeCrawler.kubeClient.NetworkingV1().Ingresses(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the ingresses networkingv1. %w", errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listIngressesNetworkingV1Beta1(namespace string) ([]networkingv1beta1.Ingress, error) {
	list, errList := kubeCrawler.kubeClient.NetworkingV1beta1().Ingresses(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the ingresses networkingv1 beta1. %w", errList)
	}

	return list.Items, nil
}
