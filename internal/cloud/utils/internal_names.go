package utils

import "fmt"

func CreateGCPInternalName(dataSourceID, zone, assetType, name string) string {
	return fmt.Sprintf("%s-%s-%s-%s", dataSourceID, zone, assetType, name)
}

func CreateKubeInternalName(dataSourceID, namespace, assetType, name string) string {
	return fmt.Sprintf("%s-%s-%s-%s", dataSourceID, namespace, assetType, name)
}

func CreateAWSInternalID(dsID string, awsElementID string) string {
	return fmt.Sprintf("%s@%s", dsID, awsElementID)
}
