package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func GetAllVpcs(region string) ([]string, error) {
	session, _ := session.NewSession(
		&aws.Config{
			Region: aws.String(region),
		},
	)

	svc := ec2.New(session)

	vpcs, errDescribeVpcs := svc.DescribeVpcs(&ec2.DescribeVpcsInput{})

	if errDescribeVpcs != nil {
		fmt.Println("Error describing the VPCs")

		return nil, fmt.Errorf("Cannot describe VPCs")
	}

	allVpcs := []string{}

	for _, vpc := range vpcs.Vpcs {
		allVpcs = append(allVpcs, *vpc.VpcId)
	}

	return allVpcs, nil
}
