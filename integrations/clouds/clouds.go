package clouds

import (
	"cleye/integrations/clouds/aws"
	"fmt"
)

// MakeCloud creates the specified cloud object
func MakeCloud(cloudType string) (interface{}, error) {
	if cloudType == "aws" {
		return aws.MakeAWS(), nil
	}

	return nil, fmt.Errorf("Unknown cloud type")
}
