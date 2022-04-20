package integrations

import (
	"time"
)

const (
	INTEGRATION_POSTGRES   = "postgres"
	INTEGRATION_AWS        = "aws"
	INTEGRATION_KUBERNETES = "kubernetes"
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
