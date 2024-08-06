package postgres

import (
	"cleye/utils"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"time"

	_ "github.com/lib/pq"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	databasemodels "dev.azure.com/bloopi/bloopi/_git/shared_models.git/database_models"
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
		SSLMode:       "disable",
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

		case "ssl_mode":
			allowedValues := []string{"require", "disable"}

			if slices.Index(allowedValues, dsConfig.Value) == -1 {
				return &crawler, fmt.Errorf("postgres config error: Value %s of config option %s is not allowed", dsConfig.Value, dsConfig.Key)
			}

			crawler.SSLMode = dsConfig.Value

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
	db, errDBConn := connectToDB(crawler.Host, crawler.User, crawler.Pass, crawler.DBName, crawler.SSLMode)
	if errDBConn != nil {
		log.Error().Msgf("Cannot connect to the Postgres db of the config %s", crawler.dataSource.DataSourceID)
		return &crawler, errDBConn
	}
	crawler.dbConn = db

	return &crawler, nil
}

func connectToDB(dbHost, dbUser, dbPass, dbName, sslMode string) (*sql.DB, error) {
	psqlConnString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=%s", dbHost, dbUser, dbPass, dbName, sslMode)

	db, err := sql.Open("postgres", psqlConnString)
	if err != nil {
		return nil, err
	}

	if errPing := db.Ping(); errPing != nil {
		return nil, errPing
	}

	db.SetMaxIdleConns(10)
	db.SetConnMaxIdleTime(1 * time.Hour)
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(20 * time.Minute)

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

	log.Info().Msgf("Starting ticker for: %s", postCrawler.dataSource.DataSourceID)
	for range crawlTicker.C {
		_, errCrawl := postCrawler.crawl()
		log.Info().Msgf("Crawling Postgres DB for %s", postCrawler.dataSource.DataSourceID)
		if errCrawl != nil {
			// do not ship any data
			log.Info().Msgf(errCrawl.Error())
			continue
		}
		// ship the crawledData to the backend
	}
}

func (postCrawler *postgresCrawler) crawl() (*bloopi_agent.CloudCrawlData, error) {
	crawlTime := time.Now().UTC()
	allCrawledElements := []*bloopi_agent.Element{}

	postDB := databasemodels.Database{
		Name:    postCrawler.DBName,
		Host:    postCrawler.Host,
		Schemas: []string{},
	}

	log.Debug().Msgf("Starting retrieval of Postgres DB schemas for %s", postCrawler.dataSource.DataSourceID)

	schemaNames, errGetSchemaNames := postCrawler.getSchemaNames()
	if errGetSchemaNames != nil {
		log.Error().Msgf("Could not retrieve the schema names because: %s", errGetSchemaNames.Error())
	}

	postDB.Schemas = schemaNames
	dbInternalName := generateInternalName(postCrawler.Host, postCrawler.DBName, "", "")
	dbElem, errDBElem := utils.CreateElement(postDB, postDB.Name, dbInternalName, post_model.POSTGRES_TYPE_DB, crawlTime)
	if errDBElem != nil {
		log.Error().Msgf("Cannot create schema db element for db name: %s because %s", postCrawler.DBName, errDBElem.Error())
		return nil, errDBElem
	}

	rel, errRel := utils.CreateRelationship(postCrawler.Host, dbInternalName, bloopi_agent.RelationshipExternalSourceSideType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
	if errRel == nil {
		allCrawledElements = append(allCrawledElements, rel)
	}

	allCrawledElements = append(allCrawledElements, dbElem)

	for _, schemaName := range schemaNames {
		schemaInternalName := generateInternalName(postCrawler.Host, postCrawler.DBName, schemaName, "")
		log.Debug().Msgf("Starting retrieval of Postgres DB schema tables for %s-%s %s", postCrawler.dataSource.Info.Type, postCrawler.dataSource.DataSourceID, schemaName)
		tableNames, errGetTableNames := postCrawler.getSchemaTables(schemaName)
		if errGetTableNames != nil {
			log.Error().Msgf("Could not get the table names for the schema %s because %s", schemaName, errGetTableNames.Error())
			continue
		}

		rel, errRel := utils.CreateRelationship(dbInternalName, schemaInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
		if errRel == nil {
			allCrawledElements = append(allCrawledElements, rel)
		}

		for _, tableName := range tableNames {
			tableInternalName := generateInternalName(postCrawler.Host, postCrawler.DBName, schemaName, tableName)
			log.Debug().Msgf("Starting retrieval of Postgres DB table columns & constraints for %s-%s %s.%s", postCrawler.dataSource.Info.Type, postCrawler.dataSource.DataSourceID, schemaName, tableName)
			table, errTable := postCrawler.getTableData(schemaName, tableName)
			if errTable != nil {
				log.Error().Msgf("Error while getting table data for table: %s.%s due to: %s", schemaName, tableName, errTable.Error())
			}
			rel, errRel := utils.CreateRelationship(schemaInternalName, tableInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
			if errRel == nil {
				allCrawledElements = append(allCrawledElements, rel)
			}

			for _, constraint := range table.Constraints {
				if constraint.Type != post_model.POSTGRES_CONSTRAINT_FK {
					continue
				}

				for _, destination := range constraint.Destinations {

					// add the referenced tableName in the current elem's relations
					rel, errRel := utils.CreateRelationship(tableInternalName, destination.Table, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
					if errRel == nil {
						allCrawledElements = append(allCrawledElements, rel)
					}
				}
			}

			log.Debug().Msgf("Starting retrieval of Postgres DB table indexes for %s-%s %s.%s", postCrawler.dataSource.Info.Type, postCrawler.dataSource.DataSourceID, schemaName, tableName)
			tableIndexes, errTableIndexes := postCrawler.getTableIndexes(schemaName, tableName)
			if errTableIndexes != nil {
				log.Info().Msgf("Cannot get the table index names for table: %s.%s because %s", schemaName, tableName, errTableIndexes.Error())
				continue
			}

			for _, tableIndex := range tableIndexes {
				indexInternalName := generateInternalName(postCrawler.Host, postCrawler.DBName, schemaName, tableIndex.Name)
				indexElem, errIndexElem := utils.CreateElement(tableIndex, tableIndex.Name, indexInternalName, post_model.POSTGRES_TYPE_INDEX, crawlTime)
				if errIndexElem != nil {
					log.Info().Msgf("Cannot create table index element for index: %s because %s", tableIndex.Name, errIndexElem.Error())
					continue
				}
				allCrawledElements = append(allCrawledElements, indexElem)
				table.Indexes = append(table.Indexes, tableIndex.Name)

				relTableIndex, errRelTableIndex := utils.CreateRelationship(tableInternalName, indexInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
				if errRelTableIndex == nil {
					allCrawledElements = append(allCrawledElements, relTableIndex)
				}

				relDBNameIndex, errRelDBNameIndex := utils.CreateRelationship(dbInternalName, indexInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
				if errRelDBNameIndex == nil {
					allCrawledElements = append(allCrawledElements, relDBNameIndex)
				}

				relSchemaIndex, errRelSchemaIndex := utils.CreateRelationship(schemaInternalName, indexInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
				if errRelSchemaIndex == nil {
					allCrawledElements = append(allCrawledElements, relSchemaIndex)
				}
			}

			tableElem, errTableElem := utils.CreateElement(table, tableName, tableInternalName, post_model.POSTGRES_TYPE_TABLE, crawlTime)
			if errTableElem != nil {
				log.Info().Msgf("Cannot create table element for table: %s because %s", tableName, errTableElem.Error())
				continue
			}
			allCrawledElements = append(allCrawledElements, tableElem)
		}

		materializedViewNames, errMaterializedViewNames := postCrawler.getSchemaMaterializedViewNames(schemaName)
		if errMaterializedViewNames != nil {
			log.Info().Msgf("Cannot get materialized view names for schema: %s because %s", schemaName, errMaterializedViewNames.Error())
			continue
		}

		for _, materializedViewName := range materializedViewNames {
			view, errView := postCrawler.getMaterializedView(schemaName, materializedViewName)
			if errView != nil {
				log.Info().Msgf("Cannot get view data for materialized view: %s because %s", materializedViewName, errView.Error())
				continue
			}

			materializedViewInternalName := generateInternalName(postCrawler.Host, postCrawler.DBName, schemaName, materializedViewName)

			rel, errRel := utils.CreateRelationship(dbInternalName, materializedViewInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
			if errRel == nil {
				allCrawledElements = append(allCrawledElements, rel)
			}

			relSchema, errRelSchema := utils.CreateRelationship(schemaInternalName, materializedViewInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
			if errRelSchema == nil {
				allCrawledElements = append(allCrawledElements, relSchema)
			}

			viewElem, errViewElem := utils.CreateElement(view, view.Name, materializedViewInternalName, post_model.POSTGRES_TYPE_MATERIALIZED_VIEW, crawlTime)
			if errViewElem != nil {
				log.Info().Msgf("Cannot create materialized view element for view: %s because %s", materializedViewName, errViewElem.Error())
				continue
			}
			allCrawledElements = append(allCrawledElements, viewElem)
		}

		schema := databasemodels.Schema{
			Name:     schemaName,
			Tables:   tableNames,
			Views:    materializedViewNames,
			Database: postDB.Name,
		}
		schemaElem, errSchemaElem := utils.CreateElement(schema, schemaName, schemaInternalName, post_model.POSTGRES_TYPE_SCHEMA, crawlTime)
		if errSchemaElem != nil {
			// We cannot process anymore if there is no schema
			continue
		}
		allCrawledElements = append(allCrawledElements, schemaElem)

		crawledData := bloopi_agent.CrawledData{
			Data: allCrawledElements,
		}

		log.Info().Msgf("Crawled %d PostgreSQL elements for connection %s and schema %s", len(allCrawledElements), postCrawler.dataSource.DataSourceID, schemaName)

		postCrawler.outputChannel <- &bloopi_agent.CloudCrawlData{
			Timestamp:       time.Now().UTC(),
			DataSource:      *postCrawler.dataSource,
			CrawledData:     crawledData,
			CrawlInternalID: schemaName,
		}
	}

	return nil, nil
}
