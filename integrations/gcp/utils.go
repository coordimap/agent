package gcp

import (
	"encoding/json"
	"fmt"
	"os"
)

func createGCPInternalName(zone string, name string) string {
	return fmt.Sprintf("%s-%s", zone, name)
}

func getZoneFromScopedZone(scopedZone string) string {
	var zone string
	fmt.Sscanf(scopedZone, "zones/%s", &zone)

	if zone == "" {
		fmt.Sscanf(scopedZone, "regions/%s", &zone)
	}

	return zone
}

// GetProjectIDFromCredentialsFile extracts project ID from a credentials file
func GetProjectIDFromCredentialsFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read credentials file: %v", err)
	}

	var key ServiceAccountKey
	if err := json.Unmarshal(data, &key); err != nil {
		return "", fmt.Errorf("failed to parse credentials file: %v", err)
	}

	if key.ProjectID == "" {
		return "", fmt.Errorf("no project_id found in credentials file")
	}

	return key.ProjectID, nil
}
