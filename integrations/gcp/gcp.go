package gcp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

func NewGCPCrawler(dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	gcpCrawler := gcpCrawler{
		clientOpts:          []option.ClientOption{},
		crawlInterval:       30 * time.Second,
		outputChan:          outChannel,
		dataSource:          *dataSource,
		InGCPEnvironment:    false,
		credentialsFile:     "",
		ConfiguredProjectID: "",
	}

	for _, config := range dataSource.Config.ValuePairs {
		switch config.Key {
		case gcpConfigInGoogleCloud:
			if strings.Compare(config.Value, "true") != 0 {
				continue
			}
			if len(gcpCrawler.clientOpts) > 0 {
				log.Info().Str("DataSourceID", gcpCrawler.dataSource.DataSourceID).Str("DataSource Type", gcpCrawler.dataSource.Info.Type).Msg("Will not take into account the credentials file as it seems that the dsta source credentials were already configured")
			}

			if !metadata.OnGCE() {
				return nil, errors.New("the agent is instructed that it will run in the Google Cloud but unfortunately no metadata server was found")
			}

			ts := google.ComputeTokenSource("")
			gcpCrawler.clientOpts = append(gcpCrawler.clientOpts, option.WithTokenSource(ts))

		case gcpConfigCredentialsFile:
			if len(gcpCrawler.clientOpts) > 0 {
				log.Info().Str("DataSourceID", gcpCrawler.dataSource.DataSourceID).Str("DataSource Type", gcpCrawler.dataSource.Info.Type).Msg("Will not take into account the credentials file as it seems that the dsta source credentials were already configured")
			}

			if _, err := os.Stat(config.Value); os.IsNotExist(err) {
				return nil, fmt.Errorf("credentials file not found: %s", config.Value)
			}
			gcpCrawler.credentialsFile = config.Value
			gcpCrawler.clientOpts = append(gcpCrawler.clientOpts, option.WithCredentialsFile(config.Value))

		case gcpProjectID:
			if config.Value == "" {
				return nil, errors.New("project_name must not be empty")
			}
			gcpCrawler.ConfiguredProjectID = strings.ToLower(config.Value)

		case gcpConfigCrawlInterval:
			amountStr := string(config.Value[:len(config.Value)-1])
			durationStr := string(config.Value[len(config.Value)-1])

			amount, errConv := strconv.ParseInt(amountStr, 10, 32)
			if errConv != nil {
				return &gcpCrawler, errConv
			}

			switch durationStr {
			case "s":
				gcpCrawler.crawlInterval = time.Duration(amount) * time.Second

			case "m":
				gcpCrawler.crawlInterval = time.Duration(amount) * time.Minute

			}
		}
	}

	credsProjectID, errCredsProjectID := gcpCrawler.GetProjectID(context.Background())
	if errCredsProjectID != nil {
		return nil, errCredsProjectID
	}
	if gcpCrawler.ConfiguredProjectID != credsProjectID {
		return nil, fmt.Errorf("the configured ProjectID %s does not match the ProjectID %s from the authentication", gcpCrawler.ConfiguredProjectID, credsProjectID)
	}

	return &gcpCrawler, nil
}

func (gcpCrawler *gcpCrawler) Crawl() {
	crawlTicker := time.NewTicker(gcpCrawler.crawlInterval)

	log.Info().Msgf("Starting ticker for: %s", gcpCrawler.dataSource.DataSourceID)
	for range crawlTicker.C {
		_, errCrawl := gcpCrawler.crawl()
		log.Info().Msgf("Crawling GCP Project for %s", gcpCrawler.dataSource.DataSourceID)
		if errCrawl != nil {
			// do not ship any data
			log.Info().Msg(errCrawl.Error())
			continue
		}
		// ship the crawledData to the backend
	}
}

func (gcpCrawler *gcpCrawler) GetProjectID(ctx context.Context) (string, error) {
	// Try to get project ID from credentials file first if available
	if gcpCrawler.credentialsFile != "" {
		projectID, err := GetProjectIDFromCredentialsFile(gcpCrawler.credentialsFile)
		if err == nil {
			return projectID, nil
		}
		// Log the error but continue with other methods
		log.Printf("Warning: Could not get project ID from credentials file: %v", err)
	}

	// Try metadata server if running on GCP
	if metadata.OnGCE() {
		projectID, err := metadata.ProjectIDWithContext(context.Background())
		if err != nil {
			return "", fmt.Errorf("failed to get project ID from metadata: %v", err)
		}
		return projectID, nil
	}

	// Try application default credentials as last resort
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("failed to get project ID from default credentials: %v", err)
	}

	if creds.ProjectID == "" {
		return "", fmt.Errorf("no project ID found in any available source")
	}

	return creds.ProjectID, nil
}

func (gcpCrawler *gcpCrawler) crawl() (*bloopi_agent.CloudCrawlData, error) {
	logger := log.With().Str("DataSourceType", "gcp").Str("ProjectID", gcpCrawler.ConfiguredProjectID).Str("DataSourceID", gcpCrawler.dataSource.DataSourceID).Logger()
	crawlTime := time.Now().UTC()
	allCrawledElemsAndRelationships := []*bloopi_agent.Element{}

	bucketElems, errBucketElems := gcpCrawler.GetBuckets(crawlTime)
	if errBucketElems != nil {
		logger.Debug().Msgf("could not retrieve buckets because %s", errBucketElems.Error())
	} else {
		allCrawledElemsAndRelationships = append(allCrawledElemsAndRelationships, bucketElems...)
	}

	cloudRunElems, errCloudRunElems := gcpCrawler.GetCloudRuns(crawlTime)
	if errCloudRunElems != nil {
		logger.Err(errCloudRunElems).Msgf("could not retrieve cloud runs.")
	} else {
		allCrawledElemsAndRelationships = append(allCrawledElemsAndRelationships, cloudRunElems...)
	}

	computeElems, errComputeElems := gcpCrawler.GetComputeElems(crawlTime)
	if errComputeElems == nil {
		allCrawledElemsAndRelationships = append(allCrawledElemsAndRelationships, computeElems...)
	}

	gkeClusterElems, errGkeClusterElems := gcpCrawler.getGKEClusters(crawlTime)
	if errGkeClusterElems == nil {
		allCrawledElemsAndRelationships = append(allCrawledElemsAndRelationships, gkeClusterElems...)
	}

	crawledData := bloopi_agent.CrawledData{
		Data: allCrawledElemsAndRelationships,
	}

	gcpCrawler.outputChan <- &bloopi_agent.CloudCrawlData{
		Timestamp:       crawlTime,
		DataSource:      gcpCrawler.dataSource,
		CrawledData:     crawledData,
		CrawlInternalID: gcpCrawler.dataSource.DataSourceID,
	}

	return nil, nil
}
