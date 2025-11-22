package mariadb

import (
	"database/sql"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
)

type mariadbCrawler struct {
	dbConn            *sql.DB
	dataSource        *bloopi_agent.DataSource
	outputChannel     chan *bloopi_agent.CloudCrawlData
	Host              string
	User              string
	Pass              string
	DBName            string
	SSLMode           string
	externalMappingID string
	crawlInterval     time.Duration
}

type Crawler interface {
	Crawl()
}
