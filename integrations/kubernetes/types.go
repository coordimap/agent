package kubernetes

import (
	"time"

	"k8s.io/client-go/kubernetes"
)

const DEFAULT_CRAWL_TIME = 30 * time.Second

const (
	KUBE_CONFIG_OPTION_IN_CLUSTER     = "in_cluster"
	KUBE_CONFIG_OPTION_CONFIG_FILE    = "config_file"
	KUBE_CONFIG_OPTION_CRAWL_INTERVAL = "crawl_interval"
)

type kubernetesCrawler struct {
	kubeClient    *kubernetes.Clientset
	crawlInterval time.Duration
}

type Crawler interface {
	Crawl()
}
