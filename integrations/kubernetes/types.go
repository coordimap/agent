package kubernetes

import (
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/prometheus/client_golang/api"
	"k8s.io/client-go/kubernetes"
)

const defaultCrawlTime = 30 * time.Second

const (
	kubeConfigInCluster            = "in_cluster"
	kubeConfigConfigFile           = "config_file"
	kubeConfigCrawlInterval        = "crawl_interval"
	kubeConfigIstioPrometheusHost  = "prometheus_host"
	kubeConfigClusterName          = "cluster_name"
	kubeConfigRetinaPrometheusHost = "retina_prometheus"
)

type kubernetesCrawler struct {
	kubeClient      *kubernetes.Clientset
	crawlInterval   time.Duration
	outputChannel   chan *bloopi_agent.CloudCrawlData
	dataSource      bloopi_agent.DataSource
	istioConfigured bool
	istioCrawler    prometheusCrawler
	clusterName     string
	retinaCrawler   *prometheusCrawler
}

type prometheusCrawler struct {
	Host          string
	promClient    api.Client
	promQueryTime string
}

type Crawler interface {
	Crawl()
}
