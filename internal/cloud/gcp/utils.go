package gcp

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

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

// ParseCloudSqlVersionToSemver takes a Cloud SQL database version string
// (e.g., POSTGRES_15, MYSQL_8_0, SQLSERVER_2019_STANDARD) and returns
// a simplified semver string (e.g., 15.0.0, 8.0.0, 2019.0.0).
// It defaults minor and patch versions to 0 if not explicitly present.
func ParseCloudSqlVersionToSemver(versionString string) (string, error) {
	if versionString == "" {
		return "", fmt.Errorf("input version string is empty")
	}

	// Use regex to capture the main version numbers, ignoring text prefixes/suffixes
	// Handles formats like:
	// POSTGRES_15 -> 15
	// MYSQL_8_0 -> 8_0
	// SQLSERVER_2019_STANDARD -> 2019
	// MYSQL_5_7 -> 5_7
	re := regexp.MustCompile(`(?:POSTGRES|MYSQL|SQLSERVER)_(\d+(?:_\d+)*).*`)
	matches := re.FindStringSubmatch(versionString)

	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse version numbers from string: %s", versionString)
	}

	// Split the captured version part by underscore
	versionParts := strings.Split(matches[1], "_")

	major := "0"
	minor := "0"
	patch := "0"

	if len(versionParts) > 0 {
		major = versionParts[0]
	}
	if len(versionParts) > 1 {
		minor = versionParts[1]
	}
	if len(versionParts) > 2 {
		patch = versionParts[2]
		// Note: Cloud SQL versions usually don't go to patch level in the enum name
		// but we handle it just in case future formats include it.
	}

	semver := fmt.Sprintf("%s.%s.%s", major, minor, patch)

	// Basic validation that the result looks like a semver (doesn't validate ranges)
	// This is a simple check, for full semver validation a dedicated library might be better.
	semverRe := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !semverRe.MatchString(semver) {
		return "", fmt.Errorf("generated version '%s' is not a valid semver format from input '%s'", semver, versionString)
	}

	return semver, nil
}
