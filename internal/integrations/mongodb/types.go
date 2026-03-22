package mongodb

import (
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"go.mongodb.org/mongo-driver/mongo"
)

type mongoCrawler struct {
	Host          string
	User          string
	Pass          string
	DBName        []string
	dbConn        *mongo.Client
	outputChannel chan *bloopi_agent.CloudCrawlData
	crawlInterval time.Duration
	dataSource    *bloopi_agent.DataSource
	scopeID       string
}

type Crawler interface {
	Crawl()
}
