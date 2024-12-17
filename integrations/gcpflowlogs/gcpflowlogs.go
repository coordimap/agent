package gcpflowlogs

import (
	"cleye/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/option"
)

func MakeGCPFlowLogsCrawler(dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	gcpFlowLogsCrawler := gcpFlowLogsCrawler{
		crawlInterval:       30 * time.Second,
		clientOpts:          []option.ClientOption{},
		credentialsFile:     "",
		dataSource:          *dataSource,
		outputChan:          outChannel,
		InGCPEnvironment:    false,
		ConfiguredProjectID: "",
	}

	for _, config := range dataSource.Config.ValuePairs {
		switch config.Key {
		case gcpFlowLogsConfigInGoogleCloud:
			if strings.Compare(config.Value, "true") != 0 {
				continue
			}
			if len(gcpFlowLogsCrawler.clientOpts) > 0 {
				log.Info().Str("DataSourceID", gcpFlowLogsCrawler.dataSource.DataSourceID).Str("DataSource Type", gcpFlowLogsCrawler.dataSource.Info.Type).Msg("Will not take into account the credentials file as it seems that the dsta source credentials were already configured")
			}

			if !metadata.OnGCE() {
				return nil, errors.New("the agent is instructed that it will run in the Google Cloud but unfortunately no metadata server was found")
			}

			ts := google.ComputeTokenSource("")
			gcpFlowLogsCrawler.clientOpts = append(gcpFlowLogsCrawler.clientOpts, option.WithTokenSource(ts))

		case gcpFlowLogsConfigCredentialsFile:
			if len(gcpFlowLogsCrawler.clientOpts) > 0 {
				log.Info().Str("DataSourceID", gcpFlowLogsCrawler.dataSource.DataSourceID).Str("DataSource Type", gcpFlowLogsCrawler.dataSource.Info.Type).Msg("Will not take into account the credentials file as it seems that the dsta source credentials were already configured")
			}

			if _, err := os.Stat(config.Value); os.IsNotExist(err) {
				return nil, fmt.Errorf("credentials file not found: %s", config.Value)
			}
			gcpFlowLogsCrawler.credentialsFile = config.Value
			gcpFlowLogsCrawler.clientOpts = append(gcpFlowLogsCrawler.clientOpts, option.WithCredentialsFile(config.Value))

		case gcpFlowLogsProjectID:
			if config.Value == "" {
				return nil, errors.New("project_name must not be empty")
			}
			gcpFlowLogsCrawler.ConfiguredProjectID = strings.ToLower(config.Value)

		case gcpFlowLogsConfigCrawlInterval:
			duration, errDuration := time.ParseDuration(config.Value)
			if errDuration != nil {
				return nil, errDuration
			}

			gcpFlowLogsCrawler.crawlInterval = duration
		}
	}

	client, errClient := logging.NewService(context.Background(), gcpFlowLogsCrawler.clientOpts...)
	if errClient != nil {
		return nil, errClient
	}

	gcpFlowLogsCrawler.client = client

	return &gcpFlowLogsCrawler, nil
}

func (crawler *gcpFlowLogsCrawler) Crawl() {
	crawlTicker := time.NewTicker(crawler.crawlInterval)

	log.Info().Msgf("Starting ticker for: %s", crawler.dataSource.DataSourceID)
	for range crawlTicker.C {
		_, errCrawl := crawler.crawl()
		log.Info().Msgf("Crawling GCP Project for Flow Logs %s", crawler.dataSource.DataSourceID)
		if errCrawl != nil {
			// do not ship any data
			log.Info().Msg(errCrawl.Error())
			continue
		}
		// ship the crawledData to the backend
	}
}

func (crawler *gcpFlowLogsCrawler) crawl() (*bloopi_agent.CloudCrawlData, error) {
	crawlTimestamp := time.Now().UTC()
	allFoundRelationships := []*bloopi_agent.Element{}
	startTime := time.Now().UTC().Add(-5 * time.Second)
	endTime := startTime.Add(crawler.crawlInterval - 5*time.Second)

	timeFilter := fmt.Sprintf(`timestamp >= "%s" AND timestamp <= "%s"`,
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339))

	filter := fmt.Sprintf(`resource.type="gce_subnetwork" AND
	                        jsonPayload.connection.src_ip!="" AND
	                        jsonPayload.connection.dest_ip!=""
	                        %s`, timeFilter)

	entries, errEntries := crawler.client.Entries.List(&logging.ListLogEntriesRequest{
		ResourceNames: []string{fmt.Sprintf("projects/%s", crawler.ConfiguredProjectID)},
		Filter:        filter,
	}).Do()
	if errEntries != nil {
		return nil, nil
	}

	for _, logEntry := range entries.Entries {
		var jsonPayload flowJSONStructure
		errUnmarshal := json.Unmarshal(logEntry.JsonPayload, &jsonPayload)

		if errUnmarshal != nil {
			return nil, errUnmarshal
		}

		crawlTime, errCrawlTime := time.Parse(time.RFC3339, jsonPayload.StartTime)
		if errCrawlTime != nil {
			crawlTime = time.Now().UTC()
		}

		if jsonPayload.SrcInstance.VmName != "" && jsonPayload.DstInstance.VmName != "" {
			srcVmInternalID := fmt.Sprintf("%s-%s", jsonPayload.SrcInstance.Zone, jsonPayload.SrcInstance.VmName)
			dstVmInternalID := fmt.Sprintf("%s-%s", jsonPayload.DstInstance.Zone, jsonPayload.DstInstance.VmName)

			vmRel, errVmRel := utils.CreateRelationship(srcVmInternalID, dstVmInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.FlowTypeRelation, crawlTime)
			if errVmRel == nil {
				allFoundRelationships = append(allFoundRelationships, vmRel)
			}
		}
	}

	crawledData := bloopi_agent.CrawledData{
		Data: allFoundRelationships,
	}

	crawler.outputChan <- &bloopi_agent.CloudCrawlData{
		Timestamp:       crawlTimestamp,
		DataSource:      crawler.dataSource,
		CrawlInternalID: crawler.dataSource.DataSourceID,
		CrawledData:     crawledData,
	}

	return nil, nil
}
