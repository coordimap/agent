package mariadb

import (
	"cleye/utils"
	"fmt"
	"slices"
	"strconv"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	databasemodels "dev.azure.com/bloopi/bloopi/_git/shared_models.git/database_models"
	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/mariadb"
	"github.com/rs/zerolog/log"
)

func NewMariadbCrawler(dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	crawler := mariadbCrawler{
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
	db, errDBConn := connectToDB(crawler.User, crawler.Pass, crawler.Host, "3306", crawler.DBName)
	if errDBConn != nil {
		log.Error().Msgf("Cannot connect to the MariaDB of the config %s", dataSource.DataSourceID)
		return &crawler, errDBConn
	}
	crawler.dbConn = db

	return &crawler, nil
}

func (mariaCrawler *mariadbCrawler) Crawl() {
	crawlTicker := time.NewTicker(mariaCrawler.crawlInterval)

	log.Info().Msgf("Starting ticker for: %s", mariaCrawler.dataSource.DataSourceID)
	for range crawlTicker.C {
		_, errCrawl := mariaCrawler.crawl()
		log.Info().Msgf("Crawling MariaDB for %s", mariaCrawler.dataSource.DataSourceID)
		if errCrawl != nil {
			// do not ship any data
			log.Info().Msgf(errCrawl.Error())
			continue
		}
		// ship the crawledData to the backend
	}
}

func (mariaCrawler *mariadbCrawler) crawl() (*bloopi_agent.CloudCrawlData, error) {
	crawlTime := time.Now().UTC()
	allCrawledElements := []*bloopi_agent.Element{}

	postDB := databasemodels.Database{
		Name:    mariaCrawler.DBName,
		Host:    mariaCrawler.Host,
		Schemas: []string{},
	}
	schemaName := mariaCrawler.DBName

	dbElem, errDBElem := utils.CreateElement(postDB, postDB.Name, postDB.Name, mariadb.MARIADB_TYPE_DB, crawlTime)
	if errDBElem != nil {
		log.Error().Msgf("Cannot create schema db element for db name: %s because %s", mariaCrawler.DBName, errDBElem.Error())
		return nil, errDBElem
	}

	allCrawledElements = append(allCrawledElements, dbElem)

	rel, errRel := utils.CreateRelationship(mariaCrawler.Host, mariaCrawler.DBName, bloopi_agent.RelationshipExternalSourceSideType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
	if errRel == nil {
		allCrawledElements = append(allCrawledElements, rel)
	}

	// get table names in schema
	tableNames, _ := mariaCrawler.GetTableNames(mariaCrawler.DBName)

	for _, tableName := range tableNames {
		internalTableName := generateInternalName(mariaCrawler.Host, schemaName, tableName)
		// get table data
		tableData, errTableData := mariaCrawler.GetTableData(schemaName, tableName)
		if errTableData != nil {
			// we need to move on because we cannot add either indexes or relationships to this specific table
			continue
		}

		// add constraints relationships
		for _, constraint := range tableData.Constraints {
			if constraint.Type != mariadb.MARIADB_CONSTRAINT_FK {
				continue
			}

			for _, destination := range constraint.Destinations {
				// add the referenced tableName in the current elem's relations
				rel, errRel := utils.CreateRelationship(internalTableName, destination.Table, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}
			}
		}

		// get table indexes
		tableIndexes, _ := mariaCrawler.getTableIndexes(schemaName, tableName)
		for _, tableIndex := range tableIndexes {
			indexInternalName := generateInternalName(mariaCrawler.Host, schemaName, tableIndex.Name)
			indexElem, errIndexElem := utils.CreateElement(tableIndex, tableIndex.Name, indexInternalName, mariadb.MARIADB_TYPE_INDEX, crawlTime)
			if errIndexElem != nil {
				log.Info().Msgf("Cannot create table index element for index: %s because %s", tableIndex.Name, errIndexElem.Error())
				continue
			}
			allCrawledElements = append(allCrawledElements, indexElem)
			tableData.Indexes = append(tableData.Indexes, indexInternalName)

			relTableIndex, errRelTableIndex := utils.CreateRelationship(internalTableName, indexInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
			if errRelTableIndex == nil {
				allCrawledElements = append(allCrawledElements, relTableIndex)
			}

			relDBNameIndex, errRelDBNameIndex := utils.CreateRelationship(mariaCrawler.DBName, indexInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ErTypeRelation, crawlTime)
			if errRelDBNameIndex == nil {
				allCrawledElements = append(allCrawledElements, relDBNameIndex)
			}
		}

		tableElem, errTableElem := utils.CreateElement(tableData, tableData.Name, internalTableName, mariadb.MARIADB_TYPE_TABLE, crawlTime)
		if errTableElem != nil {
			continue
		}
		allCrawledElements = append(allCrawledElements, tableElem)
	}

	crawledData := bloopi_agent.CrawledData{
		Data: allCrawledElements,
	}

	mariaCrawler.outputChannel <- &bloopi_agent.CloudCrawlData{
		Timestamp:       time.Now().UTC(),
		DataSource:      *mariaCrawler.dataSource,
		CrawledData:     crawledData,
		CrawlInternalID: schemaName,
	}

	return nil, nil
}
