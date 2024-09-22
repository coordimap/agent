package aws

import (
	"cleye/utils"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	aws_shared_model "dev.azure.com/bloopi/bloopi/_git/shared_models.git/aws"
	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

type AwsCrawl struct {
	ds         *bloopi_agent.DataSource
	outChannel chan *bloopi_agent.CloudCrawlData
}

// MakeAWS creates an AWS cloud struct
func MakeAWS(dsConfig *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) *AwsCrawl {
	return &AwsCrawl{
		ds:         dsConfig,
		outChannel: outChannel,
	}
}

func (awsCrawl *AwsCrawl) Crawl() {
	durationInterval, errInterval := awsCrawl.GetCrawlInterval()
	log.Info().Msgf("Ticker duration is %d seconds", durationInterval/time.Second)
	if errInterval != nil {
		// stop crawling
		log.Info().Msgf("Error in getting the interval from the configuration. %s", errInterval.Error())
		return
	}

	crawlTicker := time.NewTicker(durationInterval)

	log.Info().Msgf("Starting ticker for: %s", awsCrawl.ds.DataSourceID)
	for range crawlTicker.C {
		crawledData, errCrawl := awsCrawl.crawl()
		if errCrawl != nil {
			// do not ship any data
			log.Info().Msgf(errCrawl.Error())
			continue
		}
		// ship the crawledData to the backend
		log.Info().Msgf("Crawled %d AWS cloud elements for connection %s", len(crawledData.CrawledData.Data), awsCrawl.ds.Info.Name)
		awsCrawl.outChannel <- crawledData
	}
}

func (awsCrawl *AwsCrawl) GetCrawlInterval() (time.Duration, error) {
	for _, config := range awsCrawl.ds.Config.ValuePairs {
		if config.Key == "crawl_interval" {
			amountStr := string(config.Value[:len(config.Value)-1])
			durationStr := string(config.Value[len(config.Value)-1])

			amount, errConv := strconv.ParseInt(amountStr, 10, 32)
			if errConv != nil {
				return 0, errConv
			}

			switch durationStr {
			case "s":
				return time.Duration(amount) * time.Second, nil

			case "m":
				return time.Duration(amount) * time.Minute, nil

			default:
				return 0, fmt.Errorf("the provided duration time of %s is not one of (s, m)", durationStr)
			}
		}
	}

	return 0, errors.New("could not find crawl_interval configuration value")
}

// Crawl retrieves all the VPCs found in the specified region
// It returns a list of VPC IDs.
// 1. Create intial session and retrieve all the regions.
// 2. Loop through all the regions and store slices of each element, i.e. allVPCs
// 3. Assign all the elements to the CloudData object
// 4. return the CloudData object
func (awsCrawl *AwsCrawl) crawl() (*bloopi_agent.CloudCrawlData, error) {
	crawlTime := time.Now().UTC()
	var crawledData bloopi_agent.CrawledData

	initSession, _ := session.NewSession(
		&aws.Config{
			Region: aws.String("us-east-1"),
		},
	)

	awsRegions, errRegions := describeAllRegions(initSession, crawlTime)
	if errRegions != nil {
		return nil, fmt.Errorf("could not retrieve AWS regions")
	}

	crawledData.Data = append(crawledData.Data, awsRegions...)
	accountID, _ := getAwsAccountID(initSession)
	owner := []*string{accountID}
	results := make(chan []*bloopi_agent.Element, 5000)
	var wg sync.WaitGroup

	for _, region := range awsRegions {
		// var err error = nil
		regionSession, _ := session.NewSession(
			&aws.Config{
				Region: aws.String(region.Name),
			},
		)

		wg.Add(1)
		go worker("vpcs", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("route_tables", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("dhcp_options", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("subnets", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("natgws", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("net_acls", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("azs", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("amis", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("ec2", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("sec_groups", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("vols", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("lbs", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("lambdas", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker("rds", owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker(aws_shared_model.AwsTypeEKS, owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker(aws_shared_model.AwsTypeECRRepository, owner, regionSession, results, &wg, crawlTime)

		wg.Add(1)
		go worker(aws_shared_model.AwsTypeAutoscalingGroup, owner, regionSession, results, &wg, crawlTime)
	}

	wg.Add(1)
	go worker("s3-buckets", owner, initSession, results, &wg, crawlTime)

	ownerElement, errOwnerElement := utils.CreateElement(owner, *owner[0], *owner[0], aws_shared_model.AwsTypeOwner, crawlTime)
	if errOwnerElement == nil {
		results <- []*bloopi_agent.Element{ownerElement}
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		if len(res) != 0 {
			crawledData.Data = append(crawledData.Data, res...)
			log.Info().Msgf("Got: %s", res[0].Name)
			// log.Info().Msgf("%s  ---   %v", res[0].ID, res[0].Data)
		}
	}

	// return &crawledData, nil
	return &bloopi_agent.CloudCrawlData{
		Timestamp:       crawlTime,
		DataSource:      *awsCrawl.ds,
		CrawledData:     crawledData,
		CrawlInternalID: awsCrawl.ds.Info.Name,
	}, nil
}
