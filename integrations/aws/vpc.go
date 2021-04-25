package aws

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"cleye/integrations/clouds"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/sts"
)

// Crawl retrieves all the VPCs found in the specified region
// It returns a list of VPC IDs.
// 1. Create intial session and retrieve all the regions.
// 2. Loop through all the regions and store slices of each element, i.e. allVPCs
// 3. Assign all the elements to the CloudData object
// 4. return the CloudData object
func Crawl() (*clouds.CloudCrawlData, error) {
	var crawledData clouds.CrawledData

	initSession, _ := session.NewSession(
		&aws.Config{
			Region: aws.String("us-east-1"),
		},
	)

	awsRegions, errRegions := describeAllRegions(initSession)
	if errRegions != nil {
		return nil, fmt.Errorf("Could not retrieve AWS regions")
	}

	crawledData.Data = append(crawledData.Data, awsRegions...)
	cloudInfo, _ := getCloudAccount(initSession)
	owner := []*string{&cloudInfo.AccountID}
	results := make(chan []*clouds.Element, 5000)
	var wg sync.WaitGroup

	for _, region := range awsRegions {
		// var err error = nil
		regionSession, _ := session.NewSession(
			&aws.Config{
				Region: aws.String(region.Name),
			},
		)

		wg.Add(1)
		go worker("vpcs", owner, regionSession, results, &wg)

		wg.Add(1)
		go worker("route_tables", owner, regionSession, results, &wg)

		wg.Add(1)
		go worker("dhcp_options", owner, regionSession, results, &wg)

		wg.Add(1)
		go worker("subnets", owner, regionSession, results, &wg)

		wg.Add(1)
		go worker("natgws", owner, regionSession, results, &wg)

		wg.Add(1)
		go worker("net_acls", owner, regionSession, results, &wg)

		wg.Add(1)
		go worker("azs", owner, regionSession, results, &wg)

		wg.Add(1)
		go worker("amis", owner, regionSession, results, &wg)

		wg.Add(1)
		go worker("instances", owner, regionSession, results, &wg)

		wg.Add(1)
		go worker("sec_groups", owner, regionSession, results, &wg)

		wg.Add(1)
		go worker("vols", owner, regionSession, results, &wg)

		wg.Add(1)
		go worker("lbs", owner, regionSession, results, &wg)

		// crawledVpcs, err := describeAllVPCs(regionSession, owner)
		// crawledRouteTables, err := describeAllRouteTables(regionSession, owner)
		// dhcpOptions, err := describeAllDHCPOptions(regionSession, owner)
		// subnets, err := describeAllSubnets(regionSession, owner)
		// natgws, err := describeNATGateways(regionSession)
		// networkACLs, err := describeNetworkACLs(regionSession, owner)
		// azs, err := describeAllAvailabilityZones(regionSession)
		// amis, err := describeAllAMIs(regionSession, owner)
		// instances, err := describeAllInstances(regionSession, owner)
		// secGroups, err := describeAllSecurityGroups(regionSession, owner)
		// vols, err := describeAllVolumes(regionSession)
		// lbs, err := describeAllLoadBalancers(regionSession)

		// if err != nil {
		// 	if aerr, ok := err.(awserr.Error); ok {
		// 		switch aerr.Code() {
		// 		default:
		// 			fmt.Println(aerr.Error())
		// 		}
		// 	} else {
		// 		// Print the error, cast err to awserr.Error to get the Code and
		// 		// Message from an error.
		// 		fmt.Println(err.Error())
		// 	}
		// }

		// crawledData.Data = append(crawledData.Data, crawledVpcs...)
		// crawledData.Data = append(crawledData.Data, crawledRouteTables...)
		// crawledData.Data = append(crawledData.Data, dhcpOptions...)
		// crawledData.Data = append(crawledData.Data, subnets...)
		// crawledData.Data = append(crawledData.Data, natgws...)
		// crawledData.Data = append(crawledData.Data, networkACLs...)
		// crawledData.Data = append(crawledData.Data, azs...)
		// crawledData.Data = append(crawledData.Data, amis...)
		// crawledData.Data = append(crawledData.Data, instances...)
		// crawledData.Data = append(crawledData.Data, secGroups...)
		// crawledData.Data = append(crawledData.Data, vols...)
		// crawledData.Data = append(crawledData.Data, lbs...)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		if len(res) != 0 {
			crawledData.Data = append(crawledData.Data, res...)
			fmt.Printf("Got: %s\n", res[0].Name)
		}
	}

	marshaled, errMarshal := json.Marshal(crawledData.Data)
	if errMarshal != nil {
		return nil, errMarshal
	}

	hash, errHash := hash(marshaled)
	if errHash != nil {
		return nil, errHash
	}

	cloudData := clouds.CloudData{
		Timestamp: time.Now().UTC(),
		Data:      marshaled,
		Hash:      hash,
	}

	// return &crawledData, nil
	return &clouds.CloudCrawlData{
		CloudInfo: *cloudInfo,
		Timestamp: time.Now().UTC(),
		Data:      cloudData,
	}, nil
}

func getCloudAccount(session *session.Session) (*clouds.CloudInformation, error) {
	svc := sts.New(session)
	input := &sts.GetCallerIdentityInput{}

	result, err := svc.GetCallerIdentity(input)
	if err != nil {
		return nil, err
	}

	return &clouds.CloudInformation{
		Version:   version,
		AccountID: *result.Account,
		Name:      "aws",
		Type:      "cloud",
	}, nil
}

func describeAllVPCs(session *session.Session, owner []*string) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("owner-id"),
				Values: owner,
			},
		},
	}

	result, err := svc.DescribeVpcs(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.Vpcs {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.VpcId,
			Type:        "vpc",
			ID:          *elem.VpcId,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeAllRegions(session *session.Session) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeRegionsInput{}

	result, err := svc.DescribeRegions(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.Regions {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.RegionName,
			Type:        "regions",
			ID:          *elem.Endpoint,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeAllRouteTables(session *session.Session, owner []*string) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("owner-id"),
				Values: owner,
			},
		},
	}

	result, err := svc.DescribeRouteTables(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.RouteTables {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.RouteTableId,
			Type:        "route_table",
			ID:          *elem.RouteTableId,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeAllDHCPOptions(session *session.Session, owner []*string) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeDhcpOptionsInput{
		// DhcpOptionsIds: dhcpOptionIds,
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("owner-id"),
				Values: owner,
			},
		},
	}

	result, err := svc.DescribeDhcpOptions(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.DhcpOptions {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.DhcpOptionsId,
			Type:        "dhcp_option",
			ID:          *elem.DhcpOptionsId,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeAllSubnets(session *session.Session, owner []*string) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("owner-id"),
				Values: owner,
			},
		},
	}

	result, err := svc.DescribeSubnets(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.Subnets {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.SubnetArn,
			Type:        "subnet",
			ID:          *elem.SubnetId,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeNATGateways(session *session.Session) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeNatGatewaysInput{}

	result, err := svc.DescribeNatGateways(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.NatGateways {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.NatGatewayId,
			Type:        "natgw",
			ID:          *elem.NatGatewayId,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeNetworkACLs(session *session.Session, owner []*string) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeNetworkAclsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("owner-id"),
				Values: owner,
			},
		},
	}

	result, err := svc.DescribeNetworkAcls(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.NetworkAcls {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.NetworkAclId,
			Type:        "network_acl",
			ID:          *elem.NetworkAclId,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeAllAvailabilityZones(session *session.Session) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeAvailabilityZonesInput{}

	result, err := svc.DescribeAvailabilityZones(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.AvailabilityZones {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.ZoneName,
			Type:        "availability_zone",
			ID:          *elem.ZoneId,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeAllAMIs(session *session.Session, owner []*string) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("owner-id"),
				Values: owner,
			},
		},
	}

	result, err := svc.DescribeImages(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.Images {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.Name,
			Type:        "ami",
			ID:          *elem.ImageId,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeAllInstances(session *session.Session, owner []*string) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("owner-id"),
				Values: owner,
			},
		},
	}

	result, err := svc.DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	for _, reservation := range result.Reservations {
		for _, elem := range reservation.Instances {
			marshaled, errMarshal := json.Marshal(elem)
			if errMarshal != nil {
				continue
			}

			hash, errHash := hash(marshaled)
			if errHash != nil {
				continue
			}

			returnedElems = append(returnedElems, &clouds.Element{
				RetrievedAt: time.Now().UTC(),
				Hash:        hash,
				Name:        *elem.KeyName,
				Type:        "instance",
				ID:          *elem.InstanceId,
				Data:        marshaled,
			})
		}
	}

	return returnedElems, nil
}

func describeAllSecurityGroups(session *session.Session, owner []*string) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("owner-id"),
				Values: owner,
			},
		},
	}

	result, err := svc.DescribeSecurityGroups(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.SecurityGroups {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.GroupName,
			Type:        "security_group",
			ID:          *elem.GroupId,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeAllVolumes(session *session.Session) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := ec2.New(session)
	input := &ec2.DescribeVolumesInput{}

	result, err := svc.DescribeVolumes(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.Volumes {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.VolumeId,
			Type:        "volume",
			ID:          *elem.VolumeId,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeAllLoadBalancers(session *session.Session) ([]*clouds.Element, error) {
	var returnedElems []*clouds.Element

	svc := elbv2.New(session)
	input := &elbv2.DescribeLoadBalancersInput{}

	result, err := svc.DescribeLoadBalancers(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.LoadBalancers {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.Type,
			Type:        "classical-lb",
			ID:          *elem.LoadBalancerArn,
			Data:        marshaled,
		})
	}

	// describe classic LB
	svcElb := elb.New(session)
	inputElb := &elb.DescribeLoadBalancersInput{}

	resultElb, err := svcElb.DescribeLoadBalancers(inputElb)
	if err != nil {
		return nil, err
	}

	for _, elem := range resultElb.LoadBalancerDescriptions {
		marshaled, errMarshal := json.Marshal(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hash(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &clouds.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.LoadBalancerName,
			Type:        "lb",
			ID:          *elem.DNSName,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}
