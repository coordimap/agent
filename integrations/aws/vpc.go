package aws

import (
	"cleye/utils"
	"fmt"
	"slices"
	"strings"
	"time"

	aws_shared_model "dev.azure.com/bloopi/bloopi/_git/shared_models.git/aws"
	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/rds"
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

func describeAllVPCs(session *session.Session, owner []*string, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
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
		agentElement, _ := utils.CreateElement(elem, *elem.VpcId, *elem.VpcId, aws_shared_model.AwsTypeVpc, bloopi_agent.StatusNoStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElement)

		rel, errRel := utils.CreateRelationship(*elem.OwnerId, *elem.VpcId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}
	}

	return returnedElems, nil
}

func describeAllRegions(session *session.Session, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

	svc := ec2.New(session)
	input := &ec2.DescribeRegionsInput{}

	result, err := svc.DescribeRegions(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.Regions {
		agentElem, _ := utils.CreateElement(elem, *elem.RegionName, *elem.Endpoint, aws_shared_model.AwsTypeRegion, bloopi_agent.StatusNoStatus, "", crawlTime)

		returnedElems = append(returnedElems, agentElem)
	}

	return returnedElems, nil
}

func describeAllRouteTables(session *session.Session, owner []*string, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
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
		agentElem, _ := utils.CreateElement(elem, *elem.RouteTableId, *elem.RouteTableId, aws_shared_model.AwsTypeRouteTable, bloopi_agent.StatusNoStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)

		rel, errRel := utils.CreateRelationship(*elem.VpcId, *elem.RouteTableId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}

		rel, errRel = utils.CreateRelationship(*elem.OwnerId, *elem.RouteTableId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}
	}

	return returnedElems, nil
}

func describeAllDHCPOptions(session *session.Session, owner []*string, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
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
		agentElem, _ := utils.CreateElement(elem, *elem.DhcpOptionsId, *elem.DhcpOptionsId, aws_shared_model.AwsTypeDHCPOptions, bloopi_agent.StatusNoStatus, "", crawlTime)

		returnedElems = append(returnedElems, agentElem)
	}

	return returnedElems, nil
}

func describeAllSubnets(session *session.Session, owner []*string, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
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
		agentElem, _ := utils.CreateElement(elem, *elem.SubnetId, *elem.SubnetId, aws_shared_model.AwsTypeSubnet, bloopi_agent.StatusNoStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)

		rel, errRel := utils.CreateRelationship(*elem.VpcId, *elem.SubnetId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}

		rel, errRel = utils.CreateRelationship(*elem.OwnerId, *elem.SubnetId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}
	}

	return returnedElems, nil
}

func describeNATGateways(session *session.Session, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

	svc := ec2.New(session)
	input := &ec2.DescribeNatGatewaysInput{}

	result, err := svc.DescribeNatGateways(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.NatGateways {
		agentElem, _ := utils.CreateElement(elem, *elem.NatGatewayId, *elem.NatGatewayId, aws_shared_model.AwsTypeNatGw, bloopi_agent.StatusNoStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)

		rel, errRel := utils.CreateRelationship(*elem.VpcId, *elem.NatGatewayId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}

		rel, errRel = utils.CreateRelationship(*elem.SubnetId, *elem.NatGatewayId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}
	}

	return returnedElems, nil
}

func describeNetworkACLs(session *session.Session, owner []*string, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
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
		agentElem, _ := utils.CreateElement(elem, *elem.NetworkAclId, *elem.NetworkAclId, aws_shared_model.AwsTypeNetworkACL, bloopi_agent.StatusNoStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)

		rel, errRel := utils.CreateRelationship(*elem.VpcId, *elem.NetworkAclId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}
	}

	return returnedElems, nil
}

func describeAllAvailabilityZones(session *session.Session, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

	svc := ec2.New(session)
	input := &ec2.DescribeAvailabilityZonesInput{}

	result, err := svc.DescribeAvailabilityZones(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.AvailabilityZones {
		agentElem, _ := utils.CreateElement(elem, *elem.ZoneName, *elem.ZoneId, aws_shared_model.AwsTypeAvailabilityZone, bloopi_agent.StatusNoStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)

		rel, errRel := utils.CreateRelationship(*elem.RegionName, *elem.ZoneId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}
	}

	return returnedElems, nil
}

func describeAllAMIs(session *session.Session, owner []*string, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
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
		agentElem, _ := utils.CreateElement(elem, *elem.Name, *elem.ImageId, aws_shared_model.AwsTypeAMI, bloopi_agent.StatusNoStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)
	}

	return returnedElems, nil
}

func describeAllInstances(session *session.Session, owner []*string, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
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
			if elem.VpcId == nil || *elem.VpcId == "" {
				continue
			}

			ec2Status := bloopi_agent.StatusRed
			if *elem.State.Code == 16 {
				ec2Status = bloopi_agent.StatusGreen
			}

			agentElem, _ := utils.CreateElement(elem, *elem.InstanceId, *elem.InstanceId, aws_shared_model.AwsTypeInstance, ec2Status, "", crawlTime)
			returnedElems = append(returnedElems, agentElem)

			rel, errRel := utils.CreateRelationship(*elem.VpcId, *elem.InstanceId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}

			rel, errRel = utils.CreateRelationship(*elem.SubnetId, *elem.InstanceId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}

			rel, errRel = utils.CreateRelationship(*elem.InstanceId, *elem.ImageId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}

			for _, secGroup := range elem.SecurityGroups {
				rel, errRel := utils.CreateRelationship(*elem.InstanceId, *secGroup.GroupId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					returnedElems = append(returnedElems, rel)
				}
			}
		}
	}

	return returnedElems, nil
}

func describeAllSecurityGroups(session *session.Session, owner []*string, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
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
		agentElem, _ := utils.CreateElement(elem, *elem.GroupName, *elem.GroupId, aws_shared_model.AwsTypeSecGroup, bloopi_agent.StatusNoStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)

		rel, errRel := utils.CreateRelationship(*elem.VpcId, *elem.GroupId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}
	}

	return returnedElems, nil
}

func describeAllVolumes(session *session.Session, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

	svc := ec2.New(session)
	input := &ec2.DescribeVolumesInput{}

	result, err := svc.DescribeVolumes(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.Volumes {
		agentElem, _ := utils.CreateElement(elem, *elem.VolumeId, *elem.VolumeId, aws_shared_model.AwsTypeVolume, bloopi_agent.StatusNoStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)

		for _, volumeAttachment := range elem.Attachments {
			rel, errRel := utils.CreateRelationship(*volumeAttachment.InstanceId, *volumeAttachment.VolumeId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}
		}
	}

	return returnedElems, nil
}

func describeAllLoadBalancers(session *session.Session, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

	svc := elbv2.New(session)
	input := &elbv2.DescribeLoadBalancersInput{}

	result, err := svc.DescribeLoadBalancers(input)
	if err != nil {
		return nil, err
	}

	for _, elem := range result.LoadBalancers {
		var lbType string

		if *elem.Type == elbv2.LoadBalancerTypeEnumApplication {
			lbType = aws_shared_model.AwsTypeApplicationLoadBalancer
		} else if *elem.Type == elbv2.LoadBalancerTypeEnumNetwork {
			lbType = aws_shared_model.AwsTypeNetworkLoadBalancer
		} else if *elem.Type == elbv2.LoadBalancerTypeEnumGateway {
			lbType = aws_shared_model.AwsTypeGatewayLoadBalancer
		}

		agentElem, _ := utils.CreateElement(elem, *elem.LoadBalancerName, *elem.LoadBalancerArn, lbType, bloopi_agent.StatusNoStatus, "", crawlTime)

		for _, secGroupID := range elem.SecurityGroups {
			rel, errRel := utils.CreateRelationship(*elem.LoadBalancerArn, *secGroupID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}
		}

		rel, errRel := utils.CreateRelationship(*elem.VpcId, *elem.LoadBalancerArn, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}

		returnedElems = append(returnedElems, agentElem)

		input := &elbv2.DescribeTargetGroupsInput{
			LoadBalancerArn: elem.LoadBalancerArn,
		}
		result, err := svc.DescribeTargetGroups(input)
		if err != nil {
			continue
		}

		for _, elbTargetGroup := range result.TargetGroups {
			input := &elbv2.DescribeTargetHealthInput{
				TargetGroupArn: elbTargetGroup.TargetGroupArn,
			}
			result, err := svc.DescribeTargetHealth(input)
			if err != nil {
				continue
			}

			if *elbTargetGroup.TargetType != elbv2.TargetTypeEnumInstance {
				continue
			}

			for _, targetHealthDescription := range result.TargetHealthDescriptions {
				loadBalancerTargetRelation := bloopi_agent.RelationshipElement{
					SourceID:         *elem.LoadBalancerArn,
					DestinationID:    *targetHealthDescription.Target.Id,
					RelationshipType: aws_shared_model.AwsRelationshipTypeLoadBalancerV2Targets,
					RelationType:     bloopi_agent.ParentChildTypeRelation,
				}

				dummyID := fmt.Sprintf("%s-%s", loadBalancerTargetRelation.SourceID, loadBalancerTargetRelation.DestinationID)

				agentElem, _ := utils.CreateElement(loadBalancerTargetRelation, dummyID, dummyID, aws_shared_model.AwsTypeLoadBalancerTargetsSkipinsert, bloopi_agent.StatusNoStatus, "", crawlTime)

				// add ID-> loadbalancerarn and NAME->TargetGroupArn
				returnedElems = append(returnedElems, agentElem)
			}
		}
	}

	// describe classic LB
	svcElb := elb.New(session)
	inputElb := &elb.DescribeLoadBalancersInput{}

	resultElb, err := svcElb.DescribeLoadBalancers(inputElb)
	if err != nil {
		return nil, err
	}

	for _, elem := range resultElb.LoadBalancerDescriptions {
		agentElem, _ := utils.CreateElement(elem, *elem.LoadBalancerName, *elem.DNSName, aws_shared_model.AwsTypeClassicalLB, bloopi_agent.StatusNoStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)

		rel, errRel := utils.CreateRelationship(*elem.VPCId, *elem.DNSName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}

		for _, instance := range elem.Instances {
			rel, errRel := utils.CreateRelationship(*elem.DNSName, *instance.InstanceId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}
		}

		for _, secGroupID := range elem.SecurityGroups {
			rel, errRel := utils.CreateRelationship(*elem.DNSName, *secGroupID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}
		}
	}

	return returnedElems, nil
}

func getAllS3Buckets(session *session.Session, owner []*string, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
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
		agentElem, _ := utils.CreateElement(bucketList, *elem.Name, *elem.Name, aws_shared_model.AwsTypeS3Bucket, bloopi_agent.StatusNoStatus, "", crawlTime)

		// add relationship between the owner and the bucket
		rel, errRel := utils.CreateRelationship(*owner[0], *elem.Name, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}

		returnedElems = append(returnedElems, agentElem)
	}

	return returnedElems, nil
}

func getAllLambdaFunctions(session *session.Session, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element
	svc := lambda.New(session)

	result, err := svc.ListFunctions(nil)
	if err != nil {
		return returnedElems, err
	}

	for _, lambdaFun := range result.Functions {
		lambdaFunState := bloopi_agent.StatusNoStatus

		if lambdaFun.State != nil {
			if *lambdaFun.State == lambda.StateActive {
				lambdaFunState = bloopi_agent.StatusGreen
			} else if *lambdaFun.State == lambda.StateFailed || *lambdaFun.State == lambda.StateInactive {
				lambdaFunState = bloopi_agent.StatusRed
			} else if *lambdaFun.State == lambda.StatePending {
				lambdaFunState = bloopi_agent.StatusOrange
			}
		}

		agentElem, _ := utils.CreateElement(lambdaFun, *lambdaFun.FunctionName, *lambdaFun.FunctionArn, aws_shared_model.AwsTypeLambda, lambdaFunState, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)

		if lambdaFun.VpcConfig == nil || *lambdaFun.VpcConfig.VpcId == "" {
			continue
		}

		rel, errRel := utils.CreateRelationship(*lambdaFun.VpcConfig.VpcId, *lambdaFun.FunctionArn, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}

		for _, lambdaSubnetID := range lambdaFun.VpcConfig.SubnetIds {
			rel, errRel := utils.CreateRelationship(*lambdaSubnetID, *lambdaFun.FunctionArn, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}
		}

		for _, lambdaSecGroupID := range lambdaFun.VpcConfig.SecurityGroupIds {
			rel, errRel := utils.CreateRelationship(*lambdaSecGroupID, *lambdaFun.FunctionArn, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}
		}
	}

	return returnedElems, nil
}

func getAllRDSInstances(session *session.Session, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element
	svc := rds.New(session)

	result, err := svc.DescribeDBInstances(nil)
	if err != nil {
		return returnedElems, err
	}

	for _, dbInstance := range result.DBInstances {
		greenStates := []string{"Available", "Configuring-enhanced-monitoring", "Configuring-iam-database-auth", "Resetting-master-credentials", "Renaming"}
		redStates := []string{"Restore-error", "Storage-full", "Failed", "Deleting", "Stopped", "Stopping"}
		orangeStates := []string{}
		dbStatus := bloopi_agent.StatusNoStatus

		if slices.Contains(greenStates, *dbInstance.DBInstanceStatus) {
			dbStatus = bloopi_agent.StatusGreen
		} else if slices.Contains(redStates, *dbInstance.DBInstanceStatus) {
			dbStatus = bloopi_agent.StatusRed
		} else if slices.Contains(orangeStates, *dbInstance.DBInstanceStatus) {
			dbStatus = bloopi_agent.StatusOrange
		}

		agentElem, _ := utils.CreateElement(dbInstance, *dbInstance.Endpoint.Address, *dbInstance.Endpoint.Address, aws_shared_model.AwsTypeRDS, dbStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)

		rel, errRel := utils.CreateRelationship(*dbInstance.AvailabilityZone, *dbInstance.Endpoint.Address, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}

		for _, dbSecGroup := range dbInstance.DBSecurityGroups {
			// FIXME: this does not work. We need to get the secGroupID
			rel, errRel := utils.CreateRelationship(*dbSecGroup.DBSecurityGroupName, *dbInstance.Endpoint.Address, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}
		}

		for _, dbSecGroup := range dbInstance.VpcSecurityGroups {
			rel, errRel := utils.CreateRelationship(*dbInstance.Endpoint.Address, *dbSecGroup.VpcSecurityGroupId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}
		}

		rel, errRel = utils.CreateRelationship(*dbInstance.DBSubnetGroup.VpcId, *dbInstance.Endpoint.Address, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}

		for _, subnet := range dbInstance.DBSubnetGroup.Subnets {
			rel, errRel = utils.CreateRelationship(*subnet.SubnetIdentifier, *dbInstance.Endpoint.Address, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}

			// FIXME: this will not work as we need the AZ ID
			rel, errRel = utils.CreateRelationship(*subnet.SubnetAvailabilityZone.Name, *dbInstance.Endpoint.Address, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}
		}

	}

	return returnedElems, nil
}

func getAllAutoscalingGroups(session *session.Session, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	returnedElems := []*bloopi_agent.Element{}

	autoScalingSvc := autoscaling.New(session)
	inputDescribeAutoscalingGroup := &autoscaling.DescribeAutoScalingGroupsInput{}

	describeAutoScalingGroupsResult, errDescribeAutoscalingGroups := autoScalingSvc.DescribeAutoScalingGroups(inputDescribeAutoscalingGroup)
	if errDescribeAutoscalingGroups != nil {
		return returnedElems, errDescribeAutoscalingGroups
	}

	for _, autoScalingGroup := range describeAutoScalingGroupsResult.AutoScalingGroups {
		elem, errElem := utils.CreateElement(autoScalingGroup, *autoScalingGroup.AutoScalingGroupName, *autoScalingGroup.AutoScalingGroupARN, aws_shared_model.AwsTypeAutoscalingGroup, bloopi_agent.StatusNoStatus, "", crawlTime)
		if errElem != nil {
			continue
		}

		returnedElems = append(returnedElems, elem)

		for _, subnetID := range strings.Split(*autoScalingGroup.VPCZoneIdentifier, ",") {
			if strings.HasPrefix(*autoScalingGroup.VPCZoneIdentifier, "subnet") {
				svc := ec2.New(session)
				input := &ec2.DescribeSubnetsInput{
					Filters: []*ec2.Filter{
						{
							Name: aws.String("subnet-id"),
							Values: []*string{
								aws.String(subnetID),
							},
						},
					},
				}

				describedSubnets, errDescribeSubnet := svc.DescribeSubnets(input)
				if errDescribeSubnet == nil {
					for _, subnet := range describedSubnets.Subnets {
						rel, errRel := utils.CreateRelationship(*subnet.VpcId, subnetID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
						if errRel == nil {
							returnedElems = append(returnedElems, rel)
						}
					}
				}

				rel, errRel := utils.CreateRelationship(*autoScalingGroup.AutoScalingGroupARN, subnetID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					returnedElems = append(returnedElems, rel)
				}
			}
		}

		for _, instance := range autoScalingGroup.Instances {
			rel, errRel := utils.CreateRelationship(*autoScalingGroup.AutoScalingGroupARN, *instance.InstanceId, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}
		}

		// create link with the load balancer
		elbV2Svc := elbv2.New(session)
		input := &elbv2.DescribeLoadBalancersInput{
			Names: autoScalingGroup.LoadBalancerNames,
		}

		resultElbV2, err := elbV2Svc.DescribeLoadBalancers(input)
		if err != nil {
			return nil, err
		}

		for _, elem := range resultElbV2.LoadBalancers {
			asgInstanceRelationship, errAsgInstanceRelationship := utils.CreateRelationship(
				*autoScalingGroup.AutoScalingGroupARN,
				*elem.LoadBalancerArn,
				aws_shared_model.AwsRelationshipSkipinsert,
				aws_shared_model.AwsRelationshipSkipinsert,
				bloopi_agent.ParentChildTypeRelation,
				crawlTime,
			)
			if errAsgInstanceRelationship != nil {
				continue
			}
			returnedElems = append(returnedElems, asgInstanceRelationship)
		}

		svcElb := elb.New(session)
		inputElb := &elb.DescribeLoadBalancersInput{}

		resultElb, err := svcElb.DescribeLoadBalancers(inputElb)
		if err != nil {
			return nil, err
		}

		for _, elem := range resultElb.LoadBalancerDescriptions {
			asgInstanceRelationship, errAsgInstanceRelationship := utils.CreateRelationship(
				*autoScalingGroup.AutoScalingGroupARN,
				*elem.DNSName,
				aws_shared_model.AwsRelationshipSkipinsert,
				aws_shared_model.AwsRelationshipSkipinsert,
				bloopi_agent.ParentChildTypeRelation,
				crawlTime,
			)
			if errAsgInstanceRelationship != nil {
				continue
			}
			returnedElems = append(returnedElems, asgInstanceRelationship)
		}

		// Add the relationship between the autoscaling group and the instance id
		for _, instance := range autoScalingGroup.Instances {
			asgInstanceRelationship, errAsgInstanceRelationship := utils.CreateRelationship(
				*autoScalingGroup.AutoScalingGroupARN,
				*instance.InstanceId,
				aws_shared_model.AwsTypeAutoscalingGroup,
				aws_shared_model.AwsRelationshipSkipinsert,
				bloopi_agent.ParentChildTypeRelation,
				crawlTime,
			)
			if errAsgInstanceRelationship != nil {
				continue
			}
			returnedElems = append(returnedElems, asgInstanceRelationship)
		}
	}

	return returnedElems, nil
}

func getAllEKSClusters(session *session.Session, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element
	svc := eks.New(session)
	input := &eks.ListClustersInput{}

	result, err := svc.ListClusters(input)
	if err != nil {
		return returnedElems, err
	}

	for _, eksClusterName := range result.Clusters {
		input := &eks.DescribeClusterInput{
			Name: aws.String(*eksClusterName),
		}

		result, errDescribeCluster := svc.DescribeCluster(input)
		if errDescribeCluster != nil {
			return returnedElems, errDescribeCluster
		}

		agentElem, _ := utils.CreateElement(result.Cluster, *result.Cluster.Name, *result.Cluster.Arn, aws_shared_model.AwsTypeEKS, bloopi_agent.StatusNoStatus, "", crawlTime)
		returnedElems = append(returnedElems, agentElem)

		rel, errRel := utils.CreateRelationship(*result.Cluster.ResourcesVpcConfig.VpcId, *result.Cluster.Arn, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}

		for _, subnetID := range result.Cluster.ResourcesVpcConfig.SubnetIds {
			rel, errRel := utils.CreateRelationship(*result.Cluster.Arn, *subnetID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}
		}

		// list nodegroups of the cluster
		listNodeGroupInput := eks.ListNodegroupsInput{
			ClusterName: eksClusterName,
		}

		clusterNodeGroups, errClusterNodeGroups := svc.ListNodegroups(&listNodeGroupInput)
		if errClusterNodeGroups != nil {
			continue
		}

		for _, clusterNodeGroup := range clusterNodeGroups.Nodegroups {
			// get the nodegroup
			clusterNodeGroupInput := &eks.DescribeNodegroupInput{
				NodegroupName: clusterNodeGroup,
			}

			clusterNodeGroupInputResult, errClusterNodeGroupInput := svc.DescribeNodegroup(clusterNodeGroupInput)
			if errClusterNodeGroupInput != nil {
				continue
			}

			clusterNodeGroupElem, errClusterNodeGroupElem := utils.CreateAWSElement(
				clusterNodeGroupInputResult,
				*clusterNodeGroupInput.NodegroupName,
				*clusterNodeGroupInput.NodegroupName,
				aws_shared_model.AwsTypeEKSNodeGroup,
				bloopi_agent.StatusNoStatus,
				"",
				crawlTime,
			)
			if errClusterNodeGroupElem == nil {
				returnedElems = append(returnedElems, clusterNodeGroupElem)
			}

			rel, errRel := utils.CreateRelationship(
				*result.Cluster.Arn,
				*clusterNodeGroupInput.NodegroupName,
				bloopi_agent.RelationshipType,
				bloopi_agent.RelationshipType,
				bloopi_agent.ParentChildTypeRelation,
				crawlTime,
			)
			if errRel == nil {
				returnedElems = append(returnedElems, rel)
			}

			// get the autoscalinggroups of the nodegroup
			autoScalingSvc := autoscaling.New(session)
			autoScalingGroupNames := []*string{}
			for _, autoscalingGroup := range clusterNodeGroupInputResult.Nodegroup.Resources.AutoScalingGroups {
				autoScalingGroupNames = append(autoScalingGroupNames, autoscalingGroup.Name)
			}

			inputDescribeAutoscalingGroup := &autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: autoScalingGroupNames,
			}

			describeAutoScalingGroupsResult, errDescribeAutoscalingGroups := autoScalingSvc.DescribeAutoScalingGroups(inputDescribeAutoscalingGroup)
			if errDescribeAutoscalingGroups != nil {
				continue
			}
			for _, autoScalingGroup := range describeAutoScalingGroupsResult.AutoScalingGroups {
				relationshipAutoscalingNodegroupGroupElem, errRelationshipAutoscalingNodegroupNodeGroupElem := utils.CreateRelationship(
					*result.Cluster.Arn,
					*autoScalingGroup.AutoScalingGroupARN,
					aws_shared_model.AwsRelationshipSkipinsert,
					aws_shared_model.AwsRelationshipSkipinsert,
					bloopi_agent.ParentChildTypeRelation,
					crawlTime)
				if errRelationshipAutoscalingNodegroupNodeGroupElem != nil {
					continue
				}
				returnedElems = append(returnedElems, relationshipAutoscalingNodegroupGroupElem)
			}
		}

		// get all instances that have a tag alpha.eksctl.io/cluster-name = <EKS cluster name>
		ec2TagFilters := map[string]string{
			"alpha.eksctl.io/cluster-name": *eksClusterName,
		}

		eksInstances, errEksInstances := getFilteredEC2(session, ec2TagFilters)
		if errEksInstances != nil {
			continue
		}

		for _, eksInstance := range eksInstances {
			eksClusterInstanceRelationship, errEksClusterInstanceRelationship := utils.CreateRelationship(
				*result.Cluster.Arn,
				*eksInstance.InstanceId,
				aws_shared_model.AwsRelationshipSkipinsert,
				aws_shared_model.AwsRelationshipSkipinsert,
				bloopi_agent.ParentChildTypeRelation,
				crawlTime,
			)
			if errEksClusterInstanceRelationship != nil {
				continue
			}

			returnedElems = append(returnedElems, eksClusterInstanceRelationship)
		}
	}

	return returnedElems, nil
}

func getAllECRReposAndImages(session *session.Session, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	var returnedElems []*bloopi_agent.Element

	svc := ecr.New(session)
	input := &ecr.DescribeRepositoriesInput{}

	ecrRepos, errDescribeRepositories := svc.DescribeRepositories(input)
	if errDescribeRepositories != nil {
		return returnedElems, errDescribeRepositories
	}

	for _, ecrRepo := range ecrRepos.Repositories {

		agentElem, _ := utils.CreateElement(ecrRepo, *ecrRepo.RepositoryName, *ecrRepo.RepositoryUri, aws_shared_model.AwsTypeECRRepository, bloopi_agent.StatusNoStatus, "", crawlTime)

		returnedElems = append(returnedElems, agentElem)

		rel, errRel := utils.CreateRelationship(*ecrRepo.RegistryId, *ecrRepo.RepositoryUri, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			returnedElems = append(returnedElems, rel)
		}

		svc := ecr.New(session)
		input := &ecr.ListImagesInput{
			RepositoryName: aws.String(*ecrRepo.RepositoryName),
		}

		repoImages, errListImages := svc.ListImages(input)
		if errListImages != nil {
			continue
		}

		describeImagesInput := &ecr.DescribeImagesInput{
			ImageIds:       repoImages.ImageIds,
			RepositoryName: ecrRepo.RepositoryName,
			RegistryId:     ecrRepo.RegistryId,
		}

		describedRepoImages, errDescribedRepoImages := svc.DescribeImages(describeImagesInput)
		if errDescribedRepoImages != nil {
			continue
		}

		for _, repoImage := range describedRepoImages.ImageDetails {
			if len(repoImage.ImageTags) == 0 {
				continue
			}

			for _, imageTag := range repoImage.ImageTags {
				imageName := fmt.Sprintf("%s:%s", *ecrRepo.RepositoryUri, *imageTag)

				agentElem, _ := utils.CreateElement(repoImage, imageName, imageName, aws_shared_model.AwsTypeECRRepositoryImage, bloopi_agent.StatusNoStatus, "", crawlTime)

				returnedElems = append(returnedElems, agentElem)

				relationshipECRRepoImageElem, errRelationshipECRRepoImageElem := utils.CreateRelationship(
					*ecrRepo.RepositoryUri,
					imageName,
					aws_shared_model.AwsRelationshipSkipinsert,
					aws_shared_model.AwsRelationshipSkipinsert,
					bloopi_agent.ParentChildTypeRelation,
					crawlTime)
				if errRelationshipECRRepoImageElem == nil {
					returnedElems = append(returnedElems, relationshipECRRepoImageElem)
				}

				rel, errRel := utils.CreateRelationship(imageName, *imageTag, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					returnedElems = append(returnedElems, rel)
				}
			}

		}

		for _, repoImage := range repoImages.ImageIds {
			if repoImage.ImageTag == nil {
				continue
			}

		}

	}

	return returnedElems, nil
}
