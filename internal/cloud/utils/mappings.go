package utils

import (
	"errors"
	"fmt"
	"strings"
)

func GetMappingInternalName(configuredMappings map[string]string, mappingToSearchFor string) (string, error) {
	val, ok := configuredMappings[mappingToSearchFor]

	if !ok {
		return "", errors.New("mapping not found")
	}

	return fmt.Sprintf("%s-%s", val, mappingToSearchFor), nil
}

func GetMappingDataSourceID(configuredMappings map[string]string, mappingToSearchFor string) (string, error) {
	val, ok := configuredMappings[mappingToSearchFor]

	if !ok {
		return "", errors.New("mapping not found")
	}

	return val, nil
}

/**
* SplitConfiguredMappings
* configuredMappings is the string that is taken from the config YAML, it is of the form <internal id>@<data_source_id>. The internal id is to be formed based on the instructions in the docs.
 */
func SplitConfiguredMappings(configuredMappings string) (map[string]string, error) {
	mappings := map[string]string{}

	splitString := strings.Split(configuredMappings, " ")

	for _, mapping := range splitString {
		splitMapping := strings.Split(mapping, "@")

		if len(splitMapping) != 2 {
			continue
		}

		mappings[splitMapping[0]] = splitMapping[1]
	}

	return mappings, nil
}
