package awsflowlogs

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

func connectToAWS(region string) (*session.Session, error) {
	config := aws.Config{Region: aws.String(region)}
	sess, errSession := session.NewSession(&config)
	if errSession == nil {
		return nil, fmt.Errorf("problems with connection to AWS because %w", errSession)
	}
	return sess, nil
}
