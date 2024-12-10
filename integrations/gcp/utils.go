package gcp

import (
	"encoding/json"
	"fmt"
	"os"
)

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
