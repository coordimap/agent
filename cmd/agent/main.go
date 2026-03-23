package main

import (
	cloudutils "coordimap-agent/internal/cloud/utils"
	configuration "coordimap-agent/internal/config"
	"coordimap-agent/internal/integrations"
	"coordimap-agent/pkg/utils"
	"fmt"
	"strings"
	"time"

	"github.com/parnurzeal/gorequest"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"coordimap-agent/pkg/domain/agent"
	"coordimap-agent/pkg/domain/collector"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	endpoint   = kingpin.Flag("endpoint", "The server URL where to send data.").Default("http://localhost:8000/crawlers/infra/aws").OverrideDefaultFromEnvar("COORDIMAP_ENDPOINT").String()
	configFile = kingpin.Flag("config", "The config file path.").Default("config.yaml").OverrideDefaultFromEnvar("BLOOPIE_CONFIG_PATH").String()
	debug      = kingpin.Flag("debug", "Displays debug statements giving the user more information as to what is happening inside the agent.").Bool()
)

func getDataSourceConfigValue(dataSource *agent.DataSource, key string) string {
	for _, valuePair := range dataSource.Config.ValuePairs {
		if valuePair.Key == key {
			value, errValue := utils.LoadValueFromEnvConfig(valuePair.Value)
			if errValue == nil {
				return value
			}

			return valuePair.Value
		}
	}

	return ""
}

func validateKubernetesScopeMappings(allDataSources map[string][]*agent.DataSource) error {
	kubernetesDataSources := allDataSources[integrations.INTEGRATION_KUBERNETES]
	if len(kubernetesDataSources) == 0 {
		return nil
	}

	kubeDataSourceIDToClusterUID := map[string]string{}
	for _, dataSource := range kubernetesDataSources {
		clusterUID := getDataSourceConfigValue(dataSource, "scope_id")
		if clusterUID == "" {
			continue
		}

		kubeDataSourceIDToClusterUID[dataSource.DataSourceID] = clusterUID
	}

	if len(kubeDataSourceIDToClusterUID) == 0 {
		return nil
	}

	allValidationErrors := []string{}
	allValidationErrors = append(allValidationErrors, validateExternalMappingsForKubernetesScopes(allDataSources[integrations.INTEGRATION_GCP], integrations.INTEGRATION_GCP, kubeDataSourceIDToClusterUID)...)
	allValidationErrors = append(allValidationErrors, validateExternalMappingsForKubernetesScopes(allDataSources[integrations.INTEGRATION_EBPF_FLOWS], integrations.INTEGRATION_EBPF_FLOWS, kubeDataSourceIDToClusterUID)...)

	if len(allValidationErrors) == 0 {
		return nil
	}

	return fmt.Errorf("invalid external_mappings for kubernetes cluster scope:\n- %s", strings.Join(allValidationErrors, "\n- "))
}

func validateExternalMappingsForKubernetesScopes(dataSources []*agent.DataSource, integrationName string, kubeDataSourceIDToClusterUID map[string]string) []string {
	validationErrors := []string{}

	for _, dataSource := range dataSources {
		if integrationName == integrations.INTEGRATION_GCP {
			if getDataSourceConfigValue(dataSource, "gcp_flows") != "true" {
				continue
			}
		}

		if integrationName == integrations.INTEGRATION_EBPF_FLOWS {
			if getDataSourceConfigValue(dataSource, "deployedAt") != "kubernetes" {
				continue
			}
		}

		rawMappings := getDataSourceConfigValue(dataSource, "external_mappings")
		if rawMappings == "" {
			continue
		}

		parsedMappings, errMappings := cloudutils.SplitConfiguredMappings(rawMappings)
		if errMappings != nil {
			continue
		}

		for mappingKey, mappingValue := range parsedMappings {
			expectedClusterUID, found := kubeDataSourceIDToClusterUID[mappingValue]
			if !found {
				continue
			}

			validationErrors = append(validationErrors, fmt.Sprintf("integration=%s data_source_id=%s mapping=%s@%s expected_cluster_uid=%s", integrationName, dataSource.DataSourceID, mappingKey, mappingValue, expectedClusterUID))
		}
	}

	return validationErrors
}

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

	allDataSources := configuration.GetAllDataSources()
	errValidateMappings := validateKubernetesScopeMappings(allDataSources)
	if errValidateMappings != nil {
		log.Error().Msg(errValidateMappings.Error())
		return
	}

	sender := make(chan *agent.CloudCrawlData, 5000)

	// Steps for crawling all the configured DataSources
	// 1. Load Yaml config into the respective structs
	// 2. Loop through the configured DataSources and create the respective object
	// 		a. Configure each object with the Config specific options provided in the Yaml
	// 		b. Provide a channel to send the crawled data
	// 		c. if there is a DataSource that is not recognized, print an error and discard it
	// 3. Call Crawl() from each object to initiate crawling of the respective DataSource
	for integrationName, dss := range allDataSources {
		for _, ds := range dss {
			log.Info().Msgf("Loading crawler for %s:%s", integrationName, ds.DataSourceID)
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
			log.Error().Msgf("Cannot push data to the cloud because no data source id was found for the data source of type: %s", crawledData.DataSource.DataSourceID)
			continue
		}

		requestStruct := collector.AddCrawledInfraFromAgentRequest{
			CloudCrawlData: *crawledData,
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

		log.Info().Msgf("Sending %d elements to the collector %s for %s", len(crawledData.CrawledData.Data), *endpoint, crawledData.DataSource.DataSourceID)

		if resp.StatusCode != 200 {
			log.Error().Msgf("Could not ship any elements to the collector for data source: %s. Response was %d", crawledData.DataSource.DataSourceID, resp.StatusCode)
			continue
		}

		resp.Body.Close()
		log.Info().
			Str("CrawlTime", time.Since(crawledData.Timestamp).String()).
			Str("DataSourceID", crawledData.DataSource.DataSourceID).
			Msgf("Successfully shipped all elements.")
	}

	fmt.Println("Goodbye!!!")
}
