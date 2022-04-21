package kubernetes

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kubeCrawler *kubernetesCrawler) getNodes() ([]v1.Node, error) {
	list, err := kubeCrawler.kubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Could not load the kubernetes nodes. %w", err)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listNamespaces() ([]v1.Namespace, error) {
	list, err := kubeCrawler.kubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Could not load the kubernetes namespaces. %w", err)
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
