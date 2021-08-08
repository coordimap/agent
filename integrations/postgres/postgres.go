package postgres

import (
	"cleye/utils"
	"strconv"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/rs/zerolog/log"
)

// NewPostgresCrawler creates a new Postgresql crawler based on the input DataSource provided.
func NewPostgresCrawler(dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	// 1. initialize postgresCrawler with default values
	crawler := postgresCrawler{
		dbConn:        nil,
		outputChannel: outChannel,
		crawlInterval: 30 * time.Second,
		Host:          "localhost",
		User:          "postgres",
		Pass:          "",
		DBName:        "postgres",
		dataSource:    dataSource,
	}

	// 2. populate postgresCrawler with the provided configuration
	for _, dsConfig := range dataSource.Config.ValuePairs {
		switch dsConfig.Key {
		case "db_name":
			crawler.DBName = dsConfig.Value

		case "db_user":
			crawler.User = dsConfig.Value

		case "db_pass":
			var err error
			crawler.Pass, err = utils.LoadValueFromEnvConfig(dsConfig.Value)
			if err != nil {
				log.Info().Msgf("Error loading value of db_pass for value: %s. The error returned was: %s", dsConfig.Value, err.Error())
				return &crawler, err
			}

		case "db_host":
			crawler.Host = dsConfig.Value

		case "crawl_interval":
			numSecs, err := strconv.Atoi(dsConfig.Value)
			if err != nil {
				return &crawler, err
			}
			crawler.crawlInterval = time.Duration(numSecs) * time.Second
		}
	}

	// TODO: 3. connect to the DB

	return &crawler, nil
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
