package kubernetes

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func connectToK8sFromConfigFile(configFilePath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configFilePath)
	if err != nil {
		return nil, fmt.Errorf("Could not create Kubernetes config from file. %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Could not create Kubernetes clientset. %w", err)
	}

	return clientset, nil
}

func connectoToK8sInCluster() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("Could not create inClusterConfig.%w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Could not create Kubernetes clientSet. %w", err)
	}

	return clientset, nil
}
