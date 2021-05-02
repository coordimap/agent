package integrations

import (
	"time"
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
