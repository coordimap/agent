package kubernetes

import (
	"fmt"

	"github.com/prometheus/client_golang/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func makeIstioCrawler(prometheusHost string) (istioCrawler, error) {
	crawler := istioCrawler{
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

func generateInternalName(dataSourceID, namespace, name string) string {
	return fmt.Sprintf("%s-%s-%s", dataSourceID, namespace, name)
}
