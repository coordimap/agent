package flows

import (
	cloudutils "cleye/internal/cloud/utils"
	"cleye/utils"
	"strconv"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	kube_model "dev.azure.com/bloopi/bloopi/_git/shared_models.git/kubernetes"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func NewFlowsCrawler(dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	crawler := &flowsCrawler{
		outputChannel:     outChannel,
		dataSource:        dataSource,
		kubeClientset:     clientset,
		podCache:          NewPodCache(),
		externalMappingID: "",
	}

	for _, dsConfig := range dataSource.Config.ValuePairs {
		switch dsConfig.Key {
		case "mapping_internal_id":
			crawler.externalMappingID = dsConfig.Value

		case "crawl_interval":
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

	return crawler, nil
}

func (crawler *flowsCrawler) Crawl() {
	connectionChannel := make(chan ConnectionData, 1000)
	go MonitorNetworkTraffic(connectionChannel)

	for conn := range connectionChannel {
		srcPod := getPodInfo(crawler.kubeClientset, crawler.podCache, conn.SrcIP.String())
		dstPod := getPodInfo(crawler.kubeClientset, crawler.podCache, conn.DstIP.String())

		// Create the elements and send them to the output channel
		crawler.createAndSendElements(srcPod, dstPod, conn)
	}
}

func (crawler *flowsCrawler) createAndSendElements(srcPod, dstPod PodInfo, conn ConnectionData) {
	crawledElements := []*bloopi_agent.Element{}
	crawlTime := time.Now().UTC()

	srcInternalID := cloudutils.CreateKubeInternalName(crawler.dataSource.DataSourceID, srcPod.Namespace, kube_model.TypePod, srcPod.Name)
	dstInternalID := cloudutils.CreateKubeInternalName(crawler.dataSource.DataSourceID, dstPod.Namespace, kube_model.TypePod, dstPod.Name)

	srcElement, errSrc := utils.CreateElement(srcPod, srcPod.Name, srcInternalID, kube_model.TypePod, bloopi_agent.StatusNoStatus, "", crawlTime)
	if errSrc != nil {
		log.Warn().Msgf("Error creating source element: %s", errSrc.Error())
		return
	}

	dstElement, errDst := utils.CreateElement(dstPod, dstPod.Name, dstInternalID, kube_model.TypePod, bloopi_agent.StatusNoStatus, "", crawlTime)
	if errDst != nil {
		log.Warn().Msgf("Error creating destination element: %s", errDst.Error())
		return
	}

	relation, errRel := utils.CreateRelationship(srcInternalID, dstInternalID, "connects_to", bloopi_agent.FlowTypeRelation, crawlTime)
	if errRel != nil {
		log.Warn().Msgf("Error creating relationship: %s", errRel.Error())
		return
	}

	crawledElements = append(crawledElements, srcElement, dstElement, relation)

	crawledData := bloopi_agent.CrawledData{
		Data: crawledElements,
	}

	crawler.outputChannel <- &bloopi_agent.CloudCrawlData{
		Timestamp:       crawlTime,
		DataSource:      *crawler.dataSource,
		CrawledData:     crawledData,
		CrawlInternalID: crawler.dataSource.DataSourceID,
	}
}

func (crawler *flowsCrawler) crawl() (*bloopi_agent.CloudCrawlData, error) {
	// This function will be called by the ticker, but the main logic is now in Crawl()
	// We can leave this empty or add some periodic tasks if needed.
	return nil, nil
}
