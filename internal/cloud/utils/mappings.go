package utils

import (
	"errors"
	"fmt"
	"regexp"
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
		// Check if there is an asterisk (*) in any of the keys
		for key, value := range configuredMappings {
			newKey := strings.Replace(key, "*", ".*", 1)
			regex, errRegex := regexp.Compile(newKey)
			if errRegex != nil {
				return "", fmt.Errorf("could not create regex because %w", errRegex)
			}

			if regex.MatchString(mappingToSearchFor) {
				return value, nil
			}
		}
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

// MappingsInterface defines the interface for managing mappings.
type MappingsInterface interface {
	GetInternalName(mappingToSearchFor string) (string, error)
	GetDataSourceID(mappingToSearchFor string) (string, error)
	AddMapping(dataSourceID string, internalName string) error
	AddConfiguredMapping(configuredMappings string) error
}

// Mappings holds the configured mappings
type Mappings struct {
	mappings map[string]string
}

// NewMappings creates a new Mappings object from a raw configuration string
func NewMappings(configuredMappings string) (MappingsInterface, error) {
	m := &Mappings{mappings: make(map[string]string)}
	if err := m.AddConfiguredMapping(configuredMappings); err != nil {
		return nil, err
	}
	return m, nil
}

// GetInternalName returns the internal name for a given mapping
func (m *Mappings) GetInternalName(mappingToSearchFor string) (string, error) {
	val, ok := m.mappings[mappingToSearchFor]
	if !ok {
		return "", errors.New("mapping not found")
	}
	return fmt.Sprintf("%s-%s", val, mappingToSearchFor), nil
}

// GetDataSourceID returns the data source ID for a given mapping
func (m *Mappings) GetDataSourceID(mappingToSearchFor string) (string, error) {
	val, ok := m.mappings[mappingToSearchFor]
	if !ok {
		// Check if there is an asterisk (*) in any of the keys
		for key, value := range m.mappings {
			newKey := strings.Replace(key, "*", ".*", 1)
			regex, errRegex := regexp.Compile(newKey)
			if errRegex != nil {
				return "", fmt.Errorf("could not create regex because %w", errRegex)
			}

			if regex.MatchString(mappingToSearchFor) {
				return value, nil
			}
		}
		return "", errors.New("mapping not found")
	}
	return val, nil
}

// AddMapping adds a new mapping if it doesn't already exist.
func (m *Mappings) AddMapping(dataSourceID string, internalName string) error {
	if _, ok := m.mappings[dataSourceID]; ok {
		return fmt.Errorf("mapping for key '%s' already exists", dataSourceID)
	}
	m.mappings[dataSourceID] = internalName
	return nil
}

// AddConfiguredMapping adds new mappings from a raw configuration string.
func (m *Mappings) AddConfiguredMapping(configuredMappings string) error {
	splitString := strings.Split(configuredMappings, " ")

	for _, mapping := range splitString {
		splitMapping := strings.Split(mapping, "@")

		if len(splitMapping) != 2 {
			continue
		}

		dataSourceID := splitMapping[0]
		internalName := splitMapping[1]

		if err := m.AddMapping(dataSourceID, internalName); err != nil {
			return err
		}
	}

	return nil
}
