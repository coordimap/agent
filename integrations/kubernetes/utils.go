package kubernetes

import (
	"errors"
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/api"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func makePrometheusCrawler(prometheusHost string) (prometheusCrawler, error) {
	crawler := prometheusCrawler{
		promClient: nil,
		Host:       prometheusHost,
	}

	client, err := api.NewClient(api.Config{
		Address: prometheusHost,
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return crawler, fmt.Errorf("could not connect to the prometheus client because %w", err)
	}

	crawler.promClient = client

	return crawler, nil
}

func connectToK8sFromConfigFile(configFilePath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes config from file. %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes clientset. %w", err)
	}

	return clientset, nil
}

func connectoToK8sInCluster() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("could not create inClusterConfig.%w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes clientSet. %w", err)
	}

	return clientset, nil
}

func clearManagedFields(item *metav1.ObjectMeta) {
	item.ManagedFields = []metav1.ManagedFieldsEntry{}
}

func getNodeCloud(labels map[string]string, annotations map[string]string, addresses []v1.NodeAddress) (string, error) {
	for _, address := range addresses {
		if strings.Contains(address.Address, "compute.internal") || strings.Contains(address.Address, "amazonaws") {
			return "aws", nil
		}
	}

	for key, value := range labels {
		if strings.Contains(key, "aws") || strings.Contains(value, "aws") || strings.Contains(key, "cloud.google.com") {
			return "aws", nil
		}

		if strings.Contains(value, "google") || strings.Contains(key, "gke") || strings.Contains(value, "google") {
			return "gcp", nil
		}
	}

	for key, value := range annotations {
		if strings.Contains(key, "aws") || strings.Contains(value, "aws") {
			return "aws", nil
		}

		if strings.Contains(key, "cloud.google.com") || strings.Contains(value, "google") || strings.Contains(key, "gke") || strings.Contains(value, "google") {
			return "gcp", nil
		}
	}

	return "", errors.New("no cloud found")
}

func isNodeInCloud(labels map[string]string, annotations map[string]string, addresses []v1.NodeAddress) bool {
	for _, address := range addresses {
		if strings.Contains(address.Address, "compute.internal") || strings.Contains(address.Address, "amazonaws") {
			return true
		}
	}

	for key, value := range labels {
		if strings.Contains(key, "aws") || strings.Contains(value, "aws") || strings.Contains(key, "cloud.google.com") || strings.Contains(value, "google") || strings.Contains(key, "gke") || strings.Contains(value, "google") {
			return true
		}
	}

	for key, value := range annotations {
		if strings.Contains(key, "aws") || strings.Contains(value, "aws") || strings.Contains(key, "cloud.google.com") || strings.Contains(value, "google") || strings.Contains(key, "gke") || strings.Contains(value, "google") {
			return true
		}
	}

	return false
}
