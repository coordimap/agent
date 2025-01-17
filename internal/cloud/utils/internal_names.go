package utils

import "fmt"

func CreateGCPInternalName(dataSourceID string, zone string, name string) string {
	return fmt.Sprintf("%s-%s-%s", dataSourceID, zone, name)
}

func CreateKubeInternalName(dataSourceID, namespace, assetType, name string) string {
	return fmt.Sprintf("%s-%s-%s-%s", dataSourceID, namespace, assetType, name)
}
