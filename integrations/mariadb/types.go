package mariadb

import (
	"database/sql"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
)

type mariadbCrawler struct {
	Host          string
	User          string
	Pass          string
	DBName        string
	SSLMode       string
	dbConn        *sql.DB
	outputChannel chan *bloopi_agent.CloudCrawlData
	crawlInterval time.Duration
	dataSource    *bloopi_agent.DataSource
}

type Crawler interface {
	Crawl()
}
