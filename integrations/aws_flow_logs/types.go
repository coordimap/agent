package awsflowlogs

import (
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/aws/aws-sdk-go/aws/session"
)

const (
	AWS_FLOW_LOG_FORMAT_TYPE_DEFAULT = "default"
	AWS_FLOW_LOG_FORMAT_TYPE_ALL     = "all"
)

const (
	AWS_FLOW_LOG_FORMAT_DEFAULT = "${version} ${account-id} ${interface-id} ${srcaddr} ${dstaddr} ${srcport} ${dstport} ${protocol} ${packets} ${bytes} ${start} ${end} ${action} ${log-status}"
	AWS_FLOW_LOG_FORMAT_ALL     = "${account-id} ${action} ${az-id} ${bytes} ${dstaddr} ${dstport} ${end} ${flow-direction} ${instance-id} ${interface-id} ${log-status} ${packets} ${pkt-dst-aws-service} ${pkt-dstaddr} ${pkt-src-aws-service} ${pkt-srcaddr} ${protocol} ${region} ${srcaddr} ${srcport} ${start} ${sublocation-id} ${sublocation-type} ${subnet-id} ${tcp-flags} ${traffic-path} ${type} ${version} ${vpc-id}"
)

type awsFlowLogsCrawler struct {
	logFormat     string
	bucketName    string
	region        string
	accountID     string
	outputChannel chan *bloopi_agent.CloudCrawlData
	crawlInterval time.Duration
	dataSource    *bloopi_agent.DataSource
	awsSession    *session.Session
}

type Crawler interface {
	Crawl()
}
