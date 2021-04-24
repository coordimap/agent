package aws

import (
	"cleye/integrations/clouds"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
)

func worker(whatToCrawl string, owner []*string, regionSession *session.Session, results chan<- []*clouds.Element, wg *sync.WaitGroup) {
	defer wg.Done()

	var res []*clouds.Element
	var err error

	switch whatToCrawl {
	case "vpcs":
		res, _ = describeAllVPCs(regionSession, owner)

	case "route_tables":
		res, _ = describeAllRouteTables(regionSession, owner)

	case "dhcp_options":
		res, _ = describeAllDHCPOptions(regionSession, owner)

	case "subnets":
		res, _ = describeAllSubnets(regionSession, owner)

	case "natgws":
		res, _ = describeNATGateways(regionSession)

	case "net_acls":
		res, _ = describeNetworkACLs(regionSession, owner)

	case "azs":
		res, _ = describeAllAvailabilityZones(regionSession)

	case "amis":
		res, _ = describeAllAMIs(regionSession, owner)

	case "instances":
		res, _ = describeAllInstances(regionSession, owner)

	case "sec_groups":
		res, _ = describeAllSecurityGroups(regionSession, owner)

	case "vols":
		res, _ = describeAllVolumes(regionSession)

	case "lbs":
		res, _ = describeAllLoadBalancers(regionSession)

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
