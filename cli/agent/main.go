package main

import (
	"bytes"
	"cleye/configuration"
	"cleye/integrations"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	endpoint   = kingpin.Flag("endpoint", "The server URL where to send data.").Default("http://localhost:8000/crawlers/infra/aws").OverrideDefaultFromEnvar("CLEYE_ENDPOINT").String()
	serverKey  = kingpin.Flag("key", "The authentication API key.").Default("bloopie-test-key").OverrideDefaultFromEnvar("BLOOPIE_KEY").String()
	configFile = kingpin.Flag("config", "The config file path.").Default("config.yaml").OverrideDefaultFromEnvar("BLOOPIE_CONFIG_PATH").String()
)

func main() {
	kingpin.Version("0.1.0")
	kingpin.Parse()

	configuration := configuration.NewFileConfig(*configFile)
	log.Info().Msgf("Loading configuration file %s", *configFile)

	sender := make(chan *bloopi_agent.CloudCrawlData, 5000)

	// Steps for crawling all the configured DataSources
	// 1. Load Yaml config into the respective structs
	// 2. Loop through the configured DataSources and create the respective object
	// 		a. Configure each object with the Config specific options provided in the Yaml
	// 		b. Provide a channel to send the crawled data
	// 		c. if there is a DataSource that is not recognized, print an error and discard it
	// 3. Call Crawl() from each object to initiate crawling of the respective DataSource
	for integrationName, ds := range configuration.GetAllDataSources() {
		log.Info().Msgf("Loading crawler for %s", integrationName)
		dsCrawler, errCrawler := integrations.IntegrationsFactory(integrationName, ds, sender)
		if errCrawler != nil {
			log.Info().Msgf("Could not Crawler for integration: %s. The error was: %s", integrationName, errCrawler.Error())
			continue
		}

		go dsCrawler.Crawl()
	}

	for crawledData := range sender {
		// call the endpoint
		httpClient := http.Client{
			Timeout: 15 * time.Second,
		}

		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(crawledData)

		req, err := http.NewRequest("POST", *endpoint, b)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		req.Header.Add("BLOOPIE_KEY", *serverKey)
		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		resp.Body.Close()
	}

	fmt.Println("Goodbye!!!")
}
