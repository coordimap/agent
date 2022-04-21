package kubernetes

import (
	"context"
	"fmt"

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
