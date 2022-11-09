package mongodb

import (
	"cleye/utils"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewMongoDBCrawler(dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	// 1. initialize postgresCrawler with default values
	crawler := mongoCrawler{
		outputChannel: outChannel,
		crawlInterval: 30 * time.Second,
		Host:          "localhost",
		User:          "mongo",
		Pass:          "",
		DBName:        []string{},
		dataSource:    dataSource,
	}

	// 2. populate postgresCrawler with the provided configuration
	dbNameStar := ""
	for _, dsConfig := range dataSource.Config.ValuePairs {
		switch dsConfig.Key {
		case "db_name":
			if dsConfig.Value != "*" {
				crawler.DBName = strings.Split(dsConfig.Value, ",")
			} else {
				dbNameStar = dsConfig.Value
			}

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
	db, errDBConn := connectToDB(crawler.Host, crawler.User, crawler.Pass)
	if errDBConn != nil {
		log.Error().Msgf("Cannot connect to the Postgres db of the config %s", crawler.dataSource.Info.Name)
		return &crawler, errDBConn
	}

	// 4. in case of '*' get the names of all the databases
	if dbNameStar == "*" {
		dbNames, errListDBNames := db.ListDatabaseNames(context.Background(), bson.D{})
		if errListDBNames != nil {
			return nil, fmt.Errorf("cannot retrieve the database names because %w", errListDBNames)
		}
		crawler.DBName = dbNames
	}

	crawler.dbConn = db

	return &crawler, nil
}

func connectToDB(host, user, pass string) (*mongo.Client, error) {
	connectURI := fmt.Sprintf("mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority", user, pass, host)

	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().ApplyURI(connectURI).SetServerAPIOptions(serverAPIOptions)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (mongoCrawler *mongoCrawler) Crawl() {
	crawlTicker := time.NewTicker(mongoCrawler.crawlInterval)

	log.Info().Msgf("Starting ticker for: %s", mongoCrawler.dataSource.Info.Name)
	for range crawlTicker.C {
		_, errCrawl := mongoCrawler.crawl()
		log.Info().Msgf("Crawling Postgres DB for %s-%s", mongoCrawler.dataSource.Info.Type, mongoCrawler.dataSource.Info.Name)
		if errCrawl != nil {
			// do not ship any data
			log.Info().Msgf(errCrawl.Error())
			continue
		}
	}
}

func (mongoCrawler *mongoCrawler) crawl() (*bloopi_agent.CloudCrawlData, error) {
	crawlTime := time.Now().UTC()
	for _, dbName := range mongoCrawler.DBName {

		allCrawledElements := []*bloopi_agent.Element{}
		dbHandle := mongoCrawler.dbConn.Database(dbName)

		// get the mongo database
		mongoDB := mongoCrawler.getMongodbDatabase(dbName)
		dbElem, errDBElem := utils.CreateElement(mongoDB, mongoDB.Name, mongoDB.Name, "", crawlTime)
		if errDBElem != nil {
			return nil, errDBElem
		}
		allCrawledElements = append(allCrawledElements, dbElem)

		// TODO: get collections
		// TODO: try to infer references to other tables
		collections, errCollections := dbHandle.ListCollectionSpecifications(context.Background(), bson.D{})
		if errCollections != nil {
			return nil, errCollections
		}

		for _, collection := range collections {
			collectionHandle := dbHandle.Collection(collection.Name)
			_, errMongoCollection := mongoCrawler.getMongodbDatabaseCollection(dbHandle, collection.Name)
			if errMongoCollection != nil {
				// TODO: figure out here what to do
				continue
			}

			// TODO: get indexes
			collectionIndexes, errCollectionIndexes := mongoCrawler.listCollectionIndexes(collectionHandle)
			if errCollectionIndexes != nil {
				// TODO: log here
			}

			for _, foundIndex := range collectionIndexes {
				indexElem, errIndexElem := utils.CreateElement(foundIndex, foundIndex.Name, foundIndex.Name, "", crawlTime)
				if errIndexElem != nil {
					continue
				}
				allCrawledElements = append(allCrawledElements, indexElem)
			}
		}

		crawledData := bloopi_agent.CrawledData{
			Data: allCrawledElements,
		}

		log.Info().Msgf("Crawled %d MongoDB elements for connection %s and database %s", len(allCrawledElements), mongoCrawler.dataSource.Info.Name, dbName)

		mongoCrawler.outputChannel <- &bloopi_agent.CloudCrawlData{
			Timestamp:   time.Now().UTC(),
			DataSource:  *mongoCrawler.dataSource,
			CrawledData: crawledData,
		}
	}
	return nil, nil
}
