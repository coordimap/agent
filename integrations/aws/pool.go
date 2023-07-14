package aws

import (
	"fmt"
	"sync"
	"time"

	aws_shared_model "dev.azure.com/bloopi/bloopi/_git/shared_models.git/aws"
	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
)

func worker(whatToCrawl string, owner []*string, regionSession *session.Session, results chan<- []*bloopi_agent.Element, wg *sync.WaitGroup, crawlTime time.Time) {
	defer wg.Done()

	var res []*bloopi_agent.Element
	var err error

	switch whatToCrawl {
	case "vpcs":
		res, _ = describeAllVPCs(regionSession, owner, crawlTime)

	case "route_tables":
		res, _ = describeAllRouteTables(regionSession, owner, crawlTime)

	case "dhcp_options":
		res, _ = describeAllDHCPOptions(regionSession, owner, crawlTime)

	case "subnets":
		res, _ = describeAllSubnets(regionSession, owner, crawlTime)

	case "natgws":
		res, _ = describeNATGateways(regionSession, crawlTime)

	case "net_acls":
		res, _ = describeNetworkACLs(regionSession, owner, crawlTime)

	case "azs":
		res, _ = describeAllAvailabilityZones(regionSession, crawlTime)

	case "amis":
		res, _ = describeAllAMIs(regionSession, owner, crawlTime)

	case "ec2":
		res, _ = describeAllInstances(regionSession, owner, crawlTime)

	case "sec_groups":
		res, _ = describeAllSecurityGroups(regionSession, owner, crawlTime)

	case "vols":
		res, _ = describeAllVolumes(regionSession, crawlTime)

	case "lbs":
		res, _ = describeAllLoadBalancers(regionSession, crawlTime)

	case "s3-buckets":
		res, _ = getAllS3Buckets(regionSession, owner, crawlTime)

	case "lambdas":
		res, _ = getAllLambdaFunctions(regionSession, crawlTime)

	case "rds":
		res, _ = getAllRDSInstances(regionSession, crawlTime)

	case aws_shared_model.AwsTypeEKS:
		res, _ = getAllEKSClusters(regionSession, crawlTime)

	case aws_shared_model.AwsTypeECRRepository:
		res, _ = getAllECRReposAndImages(regionSession, crawlTime)

	case aws_shared_model.AwsTypeAutoscalingGroup:
		res, _ = getAllAutoscalingGroups(regionSession, crawlTime)

	default:
		fmt.Println("notnig")

	}

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}

	results <- res
}
