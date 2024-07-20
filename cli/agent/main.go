package main

import (
	"cleye/configuration"
	"cleye/integrations"
	"cleye/utils"
	"fmt"
	"time"

	"github.com/parnurzeal/gorequest"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/collector"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	endpoint   = kingpin.Flag("endpoint", "The server URL where to send data.").Default("http://localhost:8000/crawlers/infra/aws").OverrideDefaultFromEnvar("CLEYE_ENDPOINT").String()
	configFile = kingpin.Flag("config", "The config file path.").Default("config.yaml").OverrideDefaultFromEnvar("BLOOPIE_CONFIG_PATH").String()
	debug      = kingpin.Flag("debug", "Displays debug statements giving the user more information as to what is happening inside the agent.").Bool()
)

func main() {
	kingpin.Version("0.1.0")
	kingpin.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	configuration, errConfig := configuration.NewYamlFileConfig(*configFile)
	if errConfig != nil {
		log.Error().Msg(errConfig.Error())
		return
	}
	log.Info().Msgf("Loading configuration file %s", *configFile)

	sender := make(chan *bloopi_agent.CloudCrawlData, 5000)

	// Steps for crawling all the configured DataSources
	// 1. Load Yaml config into the respective structs
	// 2. Loop through the configured DataSources and create the respective object
	// 		a. Configure each object with the Config specific options provided in the Yaml
	// 		b. Provide a channel to send the crawled data
	// 		c. if there is a DataSource that is not recognized, print an error and discard it
	// 3. Call Crawl() from each object to initiate crawling of the respective DataSource
	for integrationName, dss := range configuration.GetAllDataSources() {
		for _, ds := range dss {
			log.Info().Msgf("Loading crawler for %s:%s", integrationName, ds.Info.Name)
			dsCrawler, errCrawler := integrations.IntegrationsFactory(integrationName, ds, sender)
			if errCrawler != nil {
				log.Info().Msgf("Could not create Crawler for integration: %s. The error was: %s", integrationName, errCrawler.Error())
				continue
			}

			go dsCrawler.Crawl()
		}
	}

	for crawledData := range sender {
		// call the endpoint

		if crawledData.DataSource.DataSourceID == "" {
			log.Error().Msgf("Cannot push data to the cloud because no data source id was found for the data source of type: %s", crawledData.DataSource.Info.Type)
			continue
		}

		requestStruct := collector.AddCrawledInfraFromAgentRequest{
			CloudCrawlData: bloopi_agent.CloudCrawlData{
				DataSource:      crawledData.DataSource,
				CrawledData:     crawledData.CrawledData,
				CrawlInternalID: crawledData.CrawlInternalID,
				Timestamp:       crawledData.Timestamp,
			},
		}

		requestStruct.CloudCrawlData.DataSource = *utils.CleanUpDataSource(&requestStruct.CloudCrawlData.DataSource, configuration.GetSkipFields())

		bloopiKey, errBloopiKey := configuration.GetCoordimapKey()
		if errBloopiKey != nil {
			log.Warn().Msg("Could not find a configurable COORDIMAP_KEY in the config file. Defaulting to 'dummy_coordimap_key'")
			bloopiKey = "dummy_bloopi_key"
		}

		var respData collector.AddCrawledInfraFromAgentResponse
		req := gorequest.New().Timeout(15 * time.Second)
		resp, _, errs := req.Post(*endpoint).Set("Api-Key", bloopiKey).SendStruct(requestStruct).EndStruct(&respData)
		if len(errs) > 0 {
			log.Info().Msgf("Error from collector %s. Error: %s", *endpoint, errs[0].Error())
			continue
		}

		if respData.Status.HTTPCode != 200 {
			log.Info().Msgf("Error from collector %s. ErrorCode: %s Error: %s", *endpoint, respData.Status.ErrorCode, respData.Status.Message)
			continue
		}

		log.Info().Msgf("Sending %d Elements to the collector %s for %s", len(crawledData.CrawledData.Data), *endpoint, crawledData.DataSource.Info.Name)

		if resp.StatusCode != 200 {
			log.Error().Msgf("Could not ship any elements to the collector. Response was %d", resp.StatusCode)
			continue
		}

		resp.Body.Close()
		log.Info().Msgf("Successfully shipped all element for %s", crawledData.DataSource.Info.Name)
	}

	fmt.Println("Goodbye!!!")
}
