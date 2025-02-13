package utils

import (
	"fmt"
	"strings"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/gcp"
	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/kubernetes"
)

func CreateGCPInternalName(dataSourceID, zone, assetType, name string) string {
	return fmt.Sprintf("%s-%s-%s-%s", dataSourceID, zone, assetType, name)
}

func CreateKubeInternalName(dataSourceID, namespace, assetType, name string) string {
	return fmt.Sprintf("%s-%s-%s-%s", dataSourceID, namespace, assetType, name)
}

func CreateAWSInternalID(dsID string, awsElementID string) string {
	return fmt.Sprintf("%s@%s", dsID, awsElementID)
}

// CreateSQLInternalName generate the internal name of the SQL server
// Examples:
// gcp:zone:name:dsid
// kube:namespace:podname:dsid
// aws:rdsname:dsid
func CreateSQLInternalName(config string) (string, error) {
	configParts := strings.Split(config, ":")
	internalName := ""

	if configParts[0] == "gcp" && len(configParts) == 4 {
		internalName = CreateGCPInternalName(configParts[3], configParts[1], gcp.TypeCloudSQL, configParts[2])
	} else if strings.HasPrefix(config, "aws") {
	} else if configParts[0] == "kube" && len(configParts) == 4 {
		internalName = CreateKubeInternalName(configParts[3], configParts[1], kubernetes.TypeNamespace, configParts[2])
	} else {
		return "", fmt.Errorf("wrong config %s", config)
	}

	return internalName, nil
}
