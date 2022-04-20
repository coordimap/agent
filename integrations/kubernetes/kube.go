package kubernetes

import (
	"cleye/utils"
	"strconv"
	"strings"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/rs/zerolog/log"
)

func MakeKubernetesCrawler(dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	clientInitialzed := false

	// Create initial kubernetesCrawler object
	crawler := kubernetesCrawler{
		kubeClient:    nil,
		crawlInterval: DEFAULT_CRAWL_TIME,
	}

	// Assign values from the config
	for _, dsConfig := range dataSource.Config.ValuePairs {
		value, errLoadValue := utils.LoadValueFromEnvConfig(dsConfig.Value)
		if errLoadValue != nil {
			log.Info().Msgf("Error loading value of db_pass for value: %s. The error returned was: %s", dsConfig.Value, errLoadValue.Error())
			return &crawler, errLoadValue
		}

		switch dsConfig.Key {

		case KUBE_CONFIG_OPTION_IN_CLUSTER:
			if strings.Compare(value, "true") != 0 || clientInitialzed {
				continue
			}

			clientSet, errClientSet := connectoToK8sInCluster()
			if errClientSet != nil {
				return &crawler, errClientSet
			}
			crawler.kubeClient = clientSet

			clientInitialzed = true

		case KUBE_CONFIG_OPTION_CONFIG_FILE:
			if clientInitialzed {
				continue
			}

			clientSet, errClientSet := connectToK8sFromConfigFile(value)
			if errClientSet != nil {
				return &crawler, errClientSet
			}

			crawler.kubeClient = clientSet

			clientInitialzed = true

		case KUBE_CONFIG_OPTION_CRAWL_INTERVAL:
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

	// Connect to the kubernetes cluster

	return crawler, nil
}

func (kubeCrawler kubernetesCrawler) Crawl() {

}
