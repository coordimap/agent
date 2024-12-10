package integrations

import (
	"cleye/integrations/aws"
	awsflowlogs "cleye/integrations/aws_flow_logs"
	"cleye/integrations/gcp"
	"cleye/integrations/kubernetes"
	"cleye/integrations/mariadb"
	"cleye/integrations/mongodb"
	"cleye/integrations/postgres"
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

	default:
		return nil, fmt.Errorf("unknown integration %s", name)
	}
}
