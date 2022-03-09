package aws

import (
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
)

func getAwsAccountID(session *session.Session) (*string, error) {
	svc := sts.New(session)
	input := &sts.GetCallerIdentityInput{}

	result, err := svc.GetCallerIdentity(input)
	if err != nil {
		return nil, err
	}

	return result.Account, nil
}

func describeAllVPCs(session *session.Session, owner []*string) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

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
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
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

func describeAllRegions(session *session.Session) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

	svc := ec2.New(session)
	input := &ec2.DescribeRegionsInput{}

	result, err := svc.DescribeRegions(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.Regions {
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.RegionName,
			Type:        "region",
			ID:          *elem.Endpoint,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeAllRouteTables(session *session.Session, owner []*string) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

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
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
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

func describeAllDHCPOptions(session *session.Session, owner []*string) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

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
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.DhcpOptionsId,
			Type:        "dhcp_options",
			ID:          *elem.DhcpOptionsId,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func describeAllSubnets(session *session.Session, owner []*string) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

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
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
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

func describeNATGateways(session *session.Session) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

	svc := ec2.New(session)
	input := &ec2.DescribeNatGatewaysInput{}

	result, err := svc.DescribeNatGateways(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.NatGateways {
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
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

func describeNetworkACLs(session *session.Session, owner []*string) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

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
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
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

func describeAllAvailabilityZones(session *session.Session) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

	svc := ec2.New(session)
	input := &ec2.DescribeAvailabilityZonesInput{}

	result, err := svc.DescribeAvailabilityZones(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.AvailabilityZones {
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
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

func describeAllAMIs(session *session.Session, owner []*string) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

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
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
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

func describeAllInstances(session *session.Session, owner []*string) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

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
			marshaled, errMarshal := encodeStruct(elem)
			if errMarshal != nil {
				continue
			}

			hash, errHash := hashGob(marshaled)
			if errHash != nil {
				continue
			}

			returnedElems = append(returnedElems, &bloopi_agent.Element{
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

func describeAllSecurityGroups(session *session.Session, owner []*string) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

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
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
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

func describeAllVolumes(session *session.Session) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

	svc := ec2.New(session)
	input := &ec2.DescribeVolumesInput{}

	result, err := svc.DescribeVolumes(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.Volumes {
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
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

func describeAllLoadBalancers(session *session.Session) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

	svc := elbv2.New(session)
	input := &elbv2.DescribeLoadBalancersInput{}

	result, err := svc.DescribeLoadBalancers(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.LoadBalancers {
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.Type,
			Type:        "load-balancer",
			ID:          *elem.DNSName,
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
		marshaled, errMarshal := encodeStruct(elem)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.LoadBalancerName,
			Type:        "classical-lb",
			ID:          *elem.DNSName,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}

func getAllS3Buckets(session *session.Session, owner []*string) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element
	svc := s3.New(session)

	result, err := svc.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	result.Owner.SetID(*owner[0])
	result.Owner.SetDisplayName(*owner[0])

	for _, elem := range result.Buckets {
		bucketList := &s3.ListBucketsOutput{
			Buckets: []*s3.Bucket{elem},
			Owner:   result.Owner,
		}
		marshaled, errMarshal := encodeStruct(bucketList)
		if errMarshal != nil {
			continue
		}

		hash, errHash := hashGob(marshaled)
		if errHash != nil {
			continue
		}

		returnedElems = append(returnedElems, &bloopi_agent.Element{
			RetrievedAt: time.Now().UTC(),
			Hash:        hash,
			Name:        *elem.Name,
			Type:        "s3-bucket",
			ID:          *elem.Name,
			Data:        marshaled,
		})
	}

	return returnedElems, nil
}
