package postgres

import "cleye/integrations"

type PostgresConfig struct {
	integrations.BaseConfig
	Host   string
	User   string
	Pass   string
	DBName string
}

type postgresCrawler struct {
	integrations.BaseConfig
	Host   string
	User   string
	Pass   string
	DBName string
}

func NewPostgresCrawler(postConfig *PostgresConfig) integrations.Crawler {
	return &postgresCrawler{}
}

// Crawl Crawls the specified Postgresql database and retrieves all the Tables/MaterializedViews
// Things that are crawled
// 1. Tables
// 2. MaterializedViews
// 3. Indexes
// 4. Relationships (foreign keys)
// 5. Sizes of Tables/Indexes/MaterializedViews
func (postCrawler *postgresCrawler) Crawl() {

}
