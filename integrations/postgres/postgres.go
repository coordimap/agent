package postgres

import (
	"cleye/utils"
	"database/sql"
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
			const DEFAULT_CRAWL_TIME = 30 * time.Second
			amountStr := string(dsConfig.Value[:len(dsConfig.Value)-1])
			durationStr := string(dsConfig.Value[len(dsConfig.Value)-1])

			amount, errConv := strconv.ParseInt(amountStr, 10, 32)
			if errConv != nil {
				return &crawler, errConv
			}

			switch durationStr {
			case "s":
				crawler.crawlInterval = time.Duration(amount) * time.Second

			case "m":
				crawler.crawlInterval = time.Duration(amount) * time.Minute

			default:
				crawler.crawlInterval = DEFAULT_CRAWL_TIME
			}
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

// Crawl Crawls the specified Postgresql database and retrieves all the Tables/MaterializedViews
// Things that are crawled
// 1. Schemas
// 2. Tables
// 3. MaterializedViews
// 4. Indexes
// 5. Relationships (foreign keys)
// 6. Sizes of Tables/Indexes/MaterializedViews
func (postCrawler *postgresCrawler) Crawl() {
	crawlTicker := time.NewTicker(postCrawler.crawlInterval)

	log.Info().Msgf("Starting ticker for: %s", postCrawler.dataSource.Info.Name)
	for range crawlTicker.C {
		crawledData, errCrawl := postCrawler.crawl()
		if errCrawl != nil {
			// do not ship any data
			log.Info().Msgf(errCrawl.Error())
			continue
		}
		// ship the crawledData to the backend
		log.Info().Msgf("Crawled %d PostgreSQL elements for connection %s", len(crawledData.CrawledData.Data), postCrawler.dataSource.Info.Name)
		postCrawler.outputChannel <- crawledData
	}
}

func (postCrawler *postgresCrawler) crawl() (*bloopi_agent.CloudCrawlData, error) {
	allCrawledElements := []*bloopi_agent.Element{}

	postDB := post_model.Database{
		Name:    postCrawler.DBName,
		Host:    postCrawler.Host,
		Schemas: []string{},
	}

	schemaNames, errGetSchemaNames := postCrawler.getSchemaNames()
	if errGetSchemaNames != nil {
		log.Error().Msgf("Could not retrieve the schema names because: %w", errGetSchemaNames)
	}

	postDB.Schemas = schemaNames
	dbElem, errDBElem := utils.CreateElement(postDB, postDB.Name, postDB.Name, post_model.POSTGRES_TYPE_DB)
	if errDBElem != nil {
		log.Error().Msgf("Cannot create schema db element for db name: %s because %w", postCrawler.DBName, errDBElem)
		return nil, errDBElem
	}

	allCrawledElements = append(allCrawledElements, dbElem)

	for _, schemaName := range schemaNames {
		tableNames, errGetTableNames := postCrawler.getSchemaTables(schemaName)
		if errGetTableNames != nil {
			log.Error().Msgf("Could not get the table names for the schema %s because %w", schemaName, errGetTableNames)
			continue
		}

		for _, tableName := range tableNames {
			table, errTable := postCrawler.getTableData(schemaName, tableName)
			if errTable != nil {
				log.Error().Msgf("Error while getting table data for table: %s due to: %w", tableName, errTable)
			}

			tableIndexes, errTableIndexes := postCrawler.getTableIndexes(schemaName, tableName)
			if errTableIndexes != nil {
				log.Info().Msgf("Cannot get the table index names for table: %s because %w", tableName, errTableIndexes)
				continue
			}

			for _, tableIndex := range tableIndexes {
				indexElem, errIndexElem := utils.CreateElement(tableIndex, tableIndex.Name, tableIndex.Name, post_model.POSTGRES_TYPE_INDEX)
				if errIndexElem != nil {
					log.Info().Msgf("Cannot create table index element for index: %s because %w", tableIndex.Name, errIndexElem)
					continue
				}
				allCrawledElements = append(allCrawledElements, indexElem)
				table.Indexes = append(table.Indexes, tableIndex.Name)
			}

			tableElem, errTableElem := utils.CreateElement(table, tableName, tableName, post_model.POSTGRES_TYPE_TABLE)
			if errTableElem != nil {
				log.Info().Msgf("Cannot create table element for table: %s because %w", tableName, errTableElem)
				continue
			}
			allCrawledElements = append(allCrawledElements, tableElem)
		}

		viewNames, errViewNames := postCrawler.getSchemaViewNames(schemaName)
		if errViewNames != nil {
			log.Info().Msgf("Cannot get view names for schema: %s because %w", schemaName, errViewNames)
			continue
		}

		for _, viewName := range viewNames {
			view, errView := postCrawler.getView(schemaName, viewName)
			if errView != nil {
				log.Info().Msgf("Cannot get view data for view: %s because %w", viewName, errView)
				continue
			}

			viewElem, errViewElem := utils.CreateElement(view, view.Name, view.Name, post_model.POSTGRES_TYPE_VIEW)
			if errViewElem != nil {
				log.Info().Msgf("Cannot create view element for view: %s because %w", viewName, errViewElem)
				continue
			}
			allCrawledElements = append(allCrawledElements, viewElem)
		}

		schema := post_model.Schema{
			Name:     schemaName,
			Tables:   tableNames,
			Views:    viewNames,
			Database: postDB.Name,
		}
		schemaElem, errSchemaElem := utils.CreateElement(schema, schemaName, schemaName, post_model.POSTGRES_TYPE_SCHEMA)
		if errSchemaElem != nil {
			// We cannot process anymore if there is no schema
			continue
		}
		allCrawledElements = append(allCrawledElements, schemaElem)
	}

	crawledData := bloopi_agent.CrawledData{
		Data: allCrawledElements,
	}

	return &bloopi_agent.CloudCrawlData{
		Timestamp:   time.Now().UTC(),
		DataSource:  *postCrawler.dataSource,
		CrawledData: crawledData,
	}, nil
}
