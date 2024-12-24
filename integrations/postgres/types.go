package postgres

import (
	"database/sql"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
)

type postgresCrawler struct {
	Host              string
	User              string
	Pass              string
	DBName            string
	SSLMode           string
	dbConn            *sql.DB
	outputChannel     chan *bloopi_agent.CloudCrawlData
	crawlInterval     time.Duration
	dataSource        *bloopi_agent.DataSource
	externalMappingID string
}

type Crawler interface {
	Crawl()
}
