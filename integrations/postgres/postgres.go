package postgres

import (
	"cleye/utils"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	_ "github.com/lib/pq"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	post_model "dev.azure.com/bloopi/bloopi/_git/shared_models.git/postgres"
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

	// 3. connect to the DB
	db, errDBConn := connectToDB(crawler.Host, crawler.User, crawler.Pass, crawler.DBName)
	if errDBConn != nil {
		log.Error().Msgf("Cannot connect to the Postgres db of the config %s", crawler.dataSource.Info.Name)
		return &crawler, errDBConn
	}
	crawler.dbConn = db

	return &crawler, nil
}

func connectToDB(dbHost, dbUser, dbPass, dbName string) (*sql.DB, error) {
	psqlConnString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbUser, dbPass, dbName)

	db, err := sql.Open("postgres", psqlConnString)
	if err != nil {
		return nil, err
	}

	if errPing := db.Ping(); errPing != nil {
		return nil, errPing
	}

	db.SetMaxIdleConns(10)
	db.SetConnMaxIdleTime(time.Hour)

	return db, nil
}

func (postCrawler *postgresCrawler) GetCrawlInterval() (time.Duration, error) {
	for _, config := range postCrawler.dataSource.Config.ValuePairs {
		if config.Key == "crawl_interval" {
			amountStr := string(config.Value[:len(config.Value)-1])
			durationStr := string(config.Value[len(config.Value)-1])

			amount, errConv := strconv.ParseInt(amountStr, 10, 32)
			if errConv != nil {
				return 0, errConv
			}

			switch durationStr {
			case "s":
				return time.Duration(amount) * time.Second, nil

			case "m":
				return time.Duration(amount) * time.Minute, nil

			default:
				return 0, fmt.Errorf("the provided duration time of %s is not one of (s, m)", durationStr)
			}
		}
	}

	return 30 * time.Second, errors.New("could not find crawl_interval configuration value, using the default 30s")
}

// Crawl Crawls the specified Postgresql database and retrieves all the Tables/MaterializedViews
// Things that are crawled
// 1. Schemas
// 2. Tables
// 3. MaterializedViews
// 4. Indexes
// 5. Relationships (foreign keys)
// 6. Sizes of Tables/Indexes/MaterializedViews
func (postCrawler *postgresCrawler) Crawl() {
	durationInterval, errInterval := postCrawler.GetCrawlInterval()
	log.Info().Msgf("Ticker duration is %d seconds", durationInterval/time.Second)
	if errInterval != nil {
		// stop crawling
		log.Info().Msgf("Error in getting the interval from the configuration. %w", errInterval)
		return
	}

	crawlTicker := time.NewTicker(durationInterval)

	log.Info().Msgf("Starting ticker for AWS: %s", postCrawler.dataSource.Info.Name)
	for range crawlTicker.C {
		crawledData, errCrawl := postCrawler.crawl()
		if errCrawl != nil {
			// do not ship any data
			log.Info().Msgf(errCrawl.Error())
			continue
		}
		// ship the crawledData to the backend
		log.Info().Msgf("Crawled %d AWS cloud elements for connection %s", len(crawledData.CrawledData.Data), postCrawler.dataSource.Info.Name)
		postCrawler.outputChannel <- crawledData
	}
}

func (postCrawler *postgresCrawler) crawl() (*bloopi_agent.CloudCrawlData, error) {
	postDB := post_model.Database{
		Name:    postCrawler.DBName,
		Host:    postCrawler.Host,
		Schemas: []post_model.Schema{},
	}

	schemaNames, errGetSchemaNames := postCrawler.getSchemaNames()
	if errGetSchemaNames != nil {
		log.Error().Msgf("Could not retrieve the schema names because: %w", errGetSchemaNames)
	}

	for _, schemaName := range schemaNames {
		schema := post_model.Schema{
			Name:   schemaName,
			Tables: []post_model.Table{},
			Views:  []post_model.View{},
		}

		tableNames, errGetTableNames := postCrawler.getSchemaTables(schemaName)
		if errGetTableNames != nil {
			log.Error().Msgf("Could not get the table names for the schema %s because %w", schemaName, errGetTableNames)
			continue
		}

		for _, tableName := range tableNames {
			table, errTable := postCrawler.getTableData(schemaName, tableName)
			if errTable != nil {
				log.Error().Msgf("Error while getting table data for table: %s due to: %w", tableName, errTable)
			} else {
				schema.Tables = append(schema.Tables, table)
			}
		}

		// TODO: Get views

		postDB.Schemas = append(postDB.Schemas, schema)
	}

	marshaled, errMarshaled := json.Marshal(postDB)
	if errMarshaled != nil {
		return nil, errMarshaled
	}

	hashed := sha256.Sum256(marshaled)

	postElem := bloopi_agent.Element{
		RetrievedAt: time.Now().UTC(),
		Hash:        hex.EncodeToString(hashed[:]),
	}

	var crawledData bloopi_agent.CrawledData

	crawledData.Data = append(crawledData.Data, &postElem)

	return &bloopi_agent.CloudCrawlData{
		Timestamp:   time.Now().UTC(),
		DataSource:  *postCrawler.dataSource,
		CrawledData: crawledData,
	}, nil
}
