package kubernetes

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
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
