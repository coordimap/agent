package aws

import (
	"cleye/integrations"
	"time"
)

type AWSConfig struct {
	integrations.BaseConfig
	InClusterConfig bool
	AccessKey       string
	Secrect         string
}

type awsCrawl struct {
	name            string
	desc            string
	crawlInterval   time.Duration
	inClusterConfig bool
	accessKey       string
	secret          string
	output          chan []string
}

// MakeAWS creates an AWS cloud struct
func MakeAWS(config *AWSConfig) integrations.Crawler {
	return &awsCrawl{
		name:            config.Name,
		desc:            config.Desc,
		crawlInterval:   config.CrawlInterval,
		inClusterConfig: config.InClusterConfig,
		accessKey:       config.AccessKey,
		secret:          config.Secrect,
		output:          config.Output,
	}
}

func (aws *awsCrawl) Crawl() {
	// TODO: implement the crawl functionality
}
