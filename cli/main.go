package main

import (
	"fmt"

	"cleye/integrations/clouds/aws"
)

func main() {
	fmt.Println(aws.GetAllVpcs("eu-central-1"))
}
