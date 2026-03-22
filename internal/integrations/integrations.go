package integrations

import (
	"coordimap-agent/internal/cloud/flows"
	"coordimap-agent/internal/cloud/gcp"
	"coordimap-agent/internal/integrations/aws"
	awsflowlogs "coordimap-agent/internal/integrations/aws_flow_logs"
	"coordimap-agent/internal/integrations/kubernetes"
	"coordimap-agent/internal/integrations/mariadb"
	"coordimap-agent/internal/integrations/mongodb"
	"coordimap-agent/internal/integrations/postgres"
	"fmt"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
)

func IntegrationsFactory(name string, dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	switch name {
	case INTEGRATION_AWS:
		return aws.MakeAWS(dataSource, outChannel), nil

	case INTEGRATION_POSTGRES:
		return postgres.NewPostgresCrawler(dataSource, outChannel)

	case INTEGRATION_KUBERNETES:
		return kubernetes.MakeKubernetesCrawler(dataSource, outChannel)

	case INTEGRATION_AWS_FLOW_LOGS:
		return awsflowlogs.NewAWSFlowLogs(dataSource, outChannel)

	case INTEGRATION_MONGODB:
		return mongodb.NewMongoDBCrawler(dataSource, outChannel)

	case INTEGRATION_MARIADB:
		return mariadb.NewMariadbCrawler(dataSource, outChannel)

	case INTEGRATION_MYSQL:
		return mariadb.NewMysqlCrawler(dataSource, outChannel)

	case INTEGRATION_GCP:
		return gcp.NewGCPCrawler(dataSource, outChannel)

	case INTEGRATION_EBPF_FLOWS:
		return flows.NewFlowsCrawler(dataSource, outChannel)

	default:
		return nil, fmt.Errorf("unknown integration %s", name)
	}
}
