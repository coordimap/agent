package aws

import (
	"cleye/utils"
	"fmt"
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
		agentElement, _ := utils.CreateElement(elem, *elem.VpcId, *elem.VpcId, aws_shared_model.AWS_TYPE_VPC, crawlTime)

		returnedElems = append(returnedElems, agentElement)
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
		agentElem, _ := utils.CreateElement(elem, *elem.RegionName, *elem.Endpoint, aws_shared_model.AWS_TYPE_REGION, crawlTime)

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
		agentElem, _ := utils.CreateElement(elem, *elem.RouteTableId, *elem.RouteTableId, aws_shared_model.AWS_TYPE_ROUTE_TABLE, crawlTime)

		returnedElems = append(returnedElems, agentElem)
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
		agentElem, _ := utils.CreateElement(elem, *elem.DhcpOptionsId, *elem.DhcpOptionsId, aws_shared_model.AWS_TYPE_DHCP_OPTIONS, crawlTime)

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
		agentElem, _ := utils.CreateElement(elem, *elem.SubnetArn, *elem.SubnetId, aws_shared_model.AWS_TYPE_SUBNET, crawlTime)

		returnedElems = append(returnedElems, agentElem)
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
		agentElem, _ := utils.CreateElement(elem, *elem.NatGatewayId, *elem.NatGatewayId, aws_shared_model.AWS_TYPE_NAT_GW, crawlTime)

		returnedElems = append(returnedElems, agentElem)
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
		agentElem, _ := utils.CreateElement(elem, *elem.NetworkAclId, *elem.NetworkAclId, aws_shared_model.AWS_TYPE_NETWORK_ACL, crawlTime)

		returnedElems = append(returnedElems, agentElem)
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
		agentElem, _ := utils.CreateElement(elem, *elem.ZoneName, *elem.ZoneId, aws_shared_model.AWS_TYPE_AVAILABILITY_ZONE, crawlTime)

		returnedElems = append(returnedElems, agentElem)
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
		agentElem, _ := utils.CreateElement(elem, *elem.Name, *elem.ImageId, aws_shared_model.AWS_TYPE_AMI, crawlTime)

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

			agentElem, _ := utils.CreateElement(elem, *elem.InstanceId, *elem.InstanceId, aws_shared_model.AWS_TYPE_INSTANCE, crawlTime)

			returnedElems = append(returnedElems, agentElem)
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
		agentElem, _ := utils.CreateElement(elem, *elem.GroupName, *elem.GroupId, aws_shared_model.AWS_TYPE_SEC_GROUP, crawlTime)

		returnedElems = append(returnedElems, agentElem)
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
		agentElem, _ := utils.CreateElement(elem, *elem.VolumeId, *elem.VolumeId, aws_shared_model.AWS_TYPE_VOLUME, crawlTime)

		returnedElems = append(returnedElems, agentElem)
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
			lbType = aws_shared_model.AWS_TYPE_APPLICATION_LOAD_BALANCER
		} else if *elem.Type == elbv2.LoadBalancerTypeEnumNetwork {
			lbType = aws_shared_model.AWS_TYPE_NETWORK_LOAD_BALANCER
		} else if *elem.Type == elbv2.LoadBalancerTypeEnumGateway {
			lbType = aws_shared_model.AWS_TYPE_GATEWAY_LOAD_BALANCER
		}

		agentElem, _ := utils.CreateElement(elem, *elem.LoadBalancerName, *elem.LoadBalancerArn, lbType, crawlTime)

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
					RelationshipType: aws_shared_model.AWS_RELATIONSHIP_TYPE_LOAD_BALANCER_V2_TARGETS,
				}

				dummyID := fmt.Sprintf("%s-%s", loadBalancerTargetRelation.SourceID, loadBalancerTargetRelation.DestinationID)

				agentElem, _ := utils.CreateElement(loadBalancerTargetRelation, dummyID, dummyID, aws_shared_model.AWS_TYPE_LOAD_BALANCER_TARGETS_SKIPINSERT, crawlTime)

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
		agentElem, _ := utils.CreateElement(elem, *elem.LoadBalancerName, *elem.DNSName, aws_shared_model.AWS_TYPE_CLASSICAL_LB, crawlTime)

		returnedElems = append(returnedElems, agentElem)
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
		agentElem, _ := utils.CreateElement(bucketList, *elem.Name, *elem.Name, aws_shared_model.AWS_TYPE_S3_BUCKET, crawlTime)

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

	for _, lambdaFunction := range result.Functions {
		if *lambdaFunction.VpcConfig.VpcId == "" {
			notConfiguredVPCID := "VPC_ID_NOT_FOUND"
			lambdaFunction.VpcConfig.VpcId = &notConfiguredVPCID
		}

		agentElem, _ := utils.CreateElement(lambdaFunction, *lambdaFunction.FunctionName, *lambdaFunction.FunctionArn, aws_shared_model.AWS_TYPE_LAMBDA, crawlTime)

		returnedElems = append(returnedElems, agentElem)
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
		agentElem, _ := utils.CreateElement(dbInstance, *dbInstance.Endpoint.Address, *dbInstance.Endpoint.Address, aws_shared_model.AWS_TYPE_RDS, crawlTime)

		returnedElems = append(returnedElems, agentElem)
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

		agentElem, _ := utils.CreateElement(result.Cluster, *result.Cluster.Name, *result.Cluster.Arn, aws_shared_model.AWS_TYPE_EKS, crawlTime)
		returnedElems = append(returnedElems, agentElem)

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

			relationshipEKSNodeGroup := bloopi_agent.RelationshipElement{
				SourceID:         *result.Cluster.Arn,
				DestinationID:    *clusterNodeGroupInputResult.Nodegroup.NodegroupArn,
				RelationshipType: aws_shared_model.AWS_RELATIONSHIP_EKS_CLUSTER_NODEGROUP,
			}

			relationshipEKSNodeGroupElem, errRelationshipEKSNodeGroupElem := utils.CreateElement(
				relationshipEKSNodeGroup,
				fmt.Sprintf("%s.%s", relationshipEKSNodeGroup.SourceID, relationshipEKSNodeGroup.DestinationID),
				fmt.Sprintf("%s.%s", relationshipEKSNodeGroup.SourceID, relationshipEKSNodeGroup.DestinationID),
				aws_shared_model.AWS_RELATIONSHIP_SKIPINSERT,
				crawlTime,
			)
			if errRelationshipEKSNodeGroupElem == nil {
				returnedElems = append(returnedElems, relationshipEKSNodeGroupElem)
			}

			clusterNodeGroupElem, errClusterNodeGroupelem := utils.CreateElement(clusterNodeGroupInputResult.Nodegroup, *clusterNodeGroupInputResult.Nodegroup.NodegroupName, *clusterNodeGroupInputResult.Nodegroup.NodegroupArn, aws_shared_model.AWS_TYPE_EKS_NODEGROUP, crawlTime)
			if errClusterNodeGroupelem != nil {
				continue
			}
			returnedElems = append(returnedElems, clusterNodeGroupElem)

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
				relationshipAutoscalingNodegroupNodeGroup := bloopi_agent.RelationshipElement{
					SourceID:         *autoScalingGroup.AutoScalingGroupARN,
					DestinationID:    *clusterNodeGroupInputResult.Nodegroup.NodegroupArn,
					RelationshipType: aws_shared_model.AWS_RELATIONSHIP_AUTOSCALING_GROUP_NODEGROUP,
				}

				relationshipAutoscalingNodegroupGroupElem, errRelationshipAutoscalingNodegroupNodeGroupElem := utils.CreateElement(
					relationshipAutoscalingNodegroupNodeGroup,
					fmt.Sprintf("%s.%s", relationshipAutoscalingNodegroupNodeGroup.SourceID, relationshipAutoscalingNodegroupNodeGroup.DestinationID),
					fmt.Sprintf("%s.%s", relationshipAutoscalingNodegroupNodeGroup.SourceID, relationshipAutoscalingNodegroupNodeGroup.DestinationID),
					aws_shared_model.AWS_RELATIONSHIP_SKIPINSERT,
					crawlTime,
				)
				if errRelationshipAutoscalingNodegroupNodeGroupElem == nil {
					returnedElems = append(returnedElems, relationshipAutoscalingNodegroupGroupElem)
				}

				elem, errElem := utils.CreateElement(autoScalingGroup, *autoScalingGroup.AutoScalingGroupName, *autoScalingGroup.AutoScalingGroupARN, aws_shared_model.AWS_TYPE_AUTOSCALING_GROUP, crawlTime)
				if errElem != nil {
					continue
				}

				returnedElems = append(returnedElems, elem)
			}
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

		agentElem, _ := utils.CreateElement(ecrRepo, *ecrRepo.RepositoryName, *ecrRepo.RepositoryUri, aws_shared_model.AWS_TYPE_ECR_REPOSITORY, crawlTime)

		returnedElems = append(returnedElems, agentElem)

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
				imageName := fmt.Sprintf("%s.%s.%s", *repoImage.RegistryId, *repoImage.RepositoryName, *imageTag)

				agentElem, _ := utils.CreateElement(repoImage, imageName, imageName, aws_shared_model.AWS_TYPE_ECR_REPOSITORY_IMAGE, crawlTime)

				returnedElems = append(returnedElems, agentElem)
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
