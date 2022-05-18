package awsflowlogs

import (
	"strconv"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
)

func NewAWSFlowLogs(dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	crawler := &awsFlowLogsCrawler{
		logFormat:     "${account-id} ${action} ${az-id} ${bytes} ${dstaddr} ${dstport} ${end} ${flow-direction} ${instance-id} ${interface-id} ${log-status} ${packets} ${pkt-dst-aws-service} ${pkt-dstaddr} ${pkt-src-aws-service} ${pkt-srcaddr} ${protocol} ${region} ${srcaddr} ${srcport} ${start} ${sublocation-id} ${sublocation-type} ${subnet-id} ${tcp-flags} ${traffic-path} ${type} ${version} ${vpc-id}",
		bucketName:    "",
		region:        "",
		outputChannel: outChannel,
		crawlInterval: 30 * time.Second,
		dataSource:    dataSource,
	}

	for _, dsConfig := range dataSource.Config.ValuePairs {
		switch dsConfig.Key {
		case "log_format":
			switch dsConfig.Value {
			case AWS_FLOW_LOG_FORMAT_DEFAULT:
				crawler.logFormat = AWS_FLOW_LOG_FORMAT_DEFAULT

			case AWS_FLOW_LOG_FORMAT_TYPE_ALL:
				crawler.logFormat = AWS_FLOW_LOG_FORMAT_ALL

			default:
				crawler.logFormat = dsConfig.Value

			}

		case "region":
			crawler.region = dsConfig.Value

		case "bucket_name":
			crawler.bucketName = dsConfig.Value

		case "account_id":
			crawler.accountID = dsConfig.Value

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

	awsSession, errAwsSession := connectToAWS(crawler.region)
	if errAwsSession != nil {
		return crawler, errAwsSession
	}

	crawler.awsSession = awsSession

	return crawler, nil
}

func (crawler *awsFlowLogsCrawler) Crawl() {

}

func (crawler *awsFlowLogsCrawler) computeStartingTimeForLogfileReading() {

}
