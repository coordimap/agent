package integrations

import (
	"cleye/integrations/aws"
	"cleye/integrations/postgres"
	"fmt"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
)

func IntegrationsFactory(name string, dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	switch name {
	case "aws":
		return aws.MakeAWS(dataSource, outChannel), nil

	case "postgres":
		return postgres.NewPostgresCrawler(dataSource, outChannel)

	default:
		return nil, fmt.Errorf("unknown integration %s", name)
	}
}
