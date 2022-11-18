package kubernetes

import (
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/prometheus/client_golang/api"
	"k8s.io/client-go/kubernetes"
)

const DEFAULT_CRAWL_TIME = 30 * time.Second

const (
	KUBE_CONFIG_OPTION_IN_CLUSTER     = "in_cluster"
	KUBE_CONFIG_OPTION_CONFIG_FILE    = "config_file"
	KUBE_CONFIG_OPTION_CRAWL_INTERVAL = "crawl_interval"
	KUBE_CONFIG_ISTIO_PROMETHEUS_HOST = "prometheus_host"
)

type kubernetesCrawler struct {
	kubeClient    *kubernetes.Clientset
	crawlInterval time.Duration
	outputChannel chan *bloopi_agent.CloudCrawlData
	dataSource    bloopi_agent.DataSource
	istioCrawler  istioCrawler
}

type istioCrawler struct {
	Host          string
	promClient    api.Client
	promQueryTime string
}

type Crawler interface {
	Crawl()
}
