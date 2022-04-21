package kubernetes

import (
	"cleye/utils"
	"strconv"
	"strings"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	kube_model "dev.azure.com/bloopi/bloopi/_git/shared_models.git/kubernetes"
	"github.com/rs/zerolog/log"
)

func MakeKubernetesCrawler(dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	clientInitialzed := false

	// Create initial kubernetesCrawler object
	crawler := &kubernetesCrawler{
		kubeClient:    nil,
		crawlInterval: DEFAULT_CRAWL_TIME,
	}

	// Assign values from the config
	for _, dsConfig := range dataSource.Config.ValuePairs {
		value, errLoadValue := utils.LoadValueFromEnvConfig(dsConfig.Value)
		if errLoadValue != nil {
			log.Info().Msgf("Error loading value of db_pass for value: %s. The error returned was: %s", dsConfig.Value, errLoadValue.Error())
			return crawler, errLoadValue
		}

		switch dsConfig.Key {

		case KUBE_CONFIG_OPTION_IN_CLUSTER:
			if strings.Compare(value, "true") != 0 || clientInitialzed {
				continue
			}

			clientSet, errClientSet := connectoToK8sInCluster()
			if errClientSet != nil {
				return crawler, errClientSet
			}
			crawler.kubeClient = clientSet

			clientInitialzed = true

		case KUBE_CONFIG_OPTION_CONFIG_FILE:
			if clientInitialzed {
				continue
			}

			clientSet, errClientSet := connectToK8sFromConfigFile(value)
			if errClientSet != nil {
				return crawler, errClientSet
			}

			crawler.kubeClient = clientSet

			clientInitialzed = true

		case KUBE_CONFIG_OPTION_CRAWL_INTERVAL:
			const DEFAULT_CRAWL_TIME = 30 * time.Second
			amountStr := string(dsConfig.Value[:len(dsConfig.Value)-1])
			durationStr := string(dsConfig.Value[len(dsConfig.Value)-1])

			amount, errConv := strconv.ParseInt(amountStr, 10, 32)
			if errConv != nil {
				return crawler, errConv
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

func (kubeCrawler *kubernetesCrawler) Crawl() {
	crawlTicker := time.NewTicker(kubeCrawler.crawlInterval)

	log.Info().Msgf("Starting ticker for AWS: %s", kubeCrawler.dataSource.Info.Name)
	for range crawlTicker.C {
		crawledData, errCrawl := kubeCrawler.crawl()
		if errCrawl != nil {
			// do not ship any data
			log.Info().Msgf(errCrawl.Error())
			continue
		}
		// ship the crawledData to the backend
		log.Info().Msgf("Crawled %d PostgreSQL elements for connection %s", len(crawledData.CrawledData.Data), kubeCrawler.dataSource.Info.Name)
		kubeCrawler.outputChannel <- crawledData
	}
}

func (kubeCrawler *kubernetesCrawler) crawl() (*bloopi_agent.CloudCrawlData, error) {
	allCrawledElements := []*bloopi_agent.Element{}

	crawledData := bloopi_agent.CrawledData{
		Data: allCrawledElements,
	}

	nodes, errNodes := kubeCrawler.getNodes()
	if errNodes != nil {
		log.Warn().Msgf("Could not get the kubernetes nodes of data source name: %s because %w", kubeCrawler.dataSource.Info.Name, errNodes)
	}

	for _, node := range nodes {
		nodeElement, errNodeElement := utils.CreateElement(node, node.Name, node.Name, kube_model.KUBERNETES_TYPE_NODE)
		if errNodeElement != nil {
			continue
		}

		allCrawledElements = append(allCrawledElements, nodeElement)
	}

	kubeNamespaces, errNamespaces := kubeCrawler.listNamespaces()
	if errNamespaces != nil {
		log.Warn().Msgf("Could not get the kubernetes namespaces of data source name: %s because %w", kubeCrawler.dataSource.Info.Name, errNamespaces)
	}

	for _, namespace := range kubeNamespaces {
		nodeElement, errNodeElement := utils.CreateElement(namespace, namespace.Name, namespace.Name, kube_model.KUBERNETES_TYPE_NAMESPACE)
		if errNodeElement != nil {
			continue
		}

		allCrawledElements = append(allCrawledElements, nodeElement)

		// get the deployments
		deployments, errDeployments := kubeCrawler.listDeplyments(namespace.Name)
		if errDeployments != nil {
			log.Warn().Msgf("Could not get the kubernetes deployments of data source name: %s because %w", kubeCrawler.dataSource.Info.Name, errDeployments)
		} else {
			for _, deployment := range deployments {
				nodeElement, errNodeElement := utils.CreateElement(deployment, deployment.Name, deployment.Name, kube_model.KUBERNETES_TYPE_DEPLOYMENT)
				if errNodeElement != nil {
					continue
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// get the services
		services, errServices := kubeCrawler.listDeplyments(namespace.Name)
		if errServices != nil {
			log.Warn().Msgf("Could not get the kubernetes services of data source name: %s because %w", kubeCrawler.dataSource.Info.Name, errServices)
		} else {
			for _, service := range services {
				nodeElement, errNodeElement := utils.CreateElement(service, service.Name, service.Name, kube_model.KUBERNETES_TYPE_SERVICE)
				if errNodeElement != nil {
					continue
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// get pods
		pods, errPods := kubeCrawler.listPods(namespace.Name)
		if errServices != nil {
			log.Warn().Msgf("Could not get the kubernetes pods of data source name: %s because %w", kubeCrawler.dataSource.Info.Name, errPods)
		} else {
			for _, pod := range pods {
				nodeElement, errNodeElement := utils.CreateElement(pod, pod.Name, pod.Name, kube_model.KUBERNETES_TYPE_POD)
				if errNodeElement != nil {
					continue
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}
	}

	return &bloopi_agent.CloudCrawlData{
		Timestamp:   time.Now().UTC(),
		DataSource:  kubeCrawler.dataSource,
		CrawledData: crawledData,
	}, nil
}
