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
	kubeConfigCloudDataSourceID    = "cloud_data_source_id"
)

type kubernetesCrawler struct {
	retinaCrawler     *prometheusCrawler
	kubeClient        *kubernetes.Clientset
	outputChannel     chan *bloopi_agent.CloudCrawlData
	dataSource        bloopi_agent.DataSource
	istioCrawler      prometheusCrawler
	internalNodeNames map[string]string
	clusterName       string
	cloudDataSourceID string
	istioConfigured   bool
	crawlInterval     time.Duration
}

type prometheusCrawler struct {
	Host          string
	promClient    api.Client
	promQueryTime string
}

type Crawler interface {
	Crawl()
}
