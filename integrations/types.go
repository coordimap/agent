package integrations

import (
	"time"
)

const (
	INTEGRATION_POSTGRES      = "postgres"
	INTEGRATION_AWS           = "aws"
	INTEGRATION_KUBERNETES    = "kubernetes"
	INTEGRATION_AWS_FLOW_LOGS = "aws_flow_logs"
	INTEGRATION_MONGODB       = "mongodb"
	INTEGRATION_MARIADB       = "mariadb"
	INTEGRATION_MYSQL         = "mysql"
	INTEGRATION_GCP           = "gcp"
	INTEGRATION_GCP_FLOW_LOGS = "gcp_flow_logs"
)

type BaseConfig struct {
	CrawlInterval time.Duration
	Name          string
	Desc          string
	Output        chan []string
}

type Crawler interface {
	Crawl()
}
