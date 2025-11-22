package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
)

func LoadValueFromEnvConfig(value string) (string, error) {
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		envVarName := value[2 : len(value)-1]
		envValue, exists := os.LookupEnv(envVarName)
		if !exists {
			return "", fmt.Errorf("could not find an environment variable for the Value: %s", value)
		}

		return envValue, nil
	}

	return value, nil
}

func encodeAndHashAWSStruct(elem interface{}) ([]byte, string, error) {
	var buff bytes.Buffer
	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(elem)
	marshaledElem := buff.Bytes()
	buff.Reset()

	hashArr := sha256.Sum256(marshaledElem)
	hashStr := hex.EncodeToString(hashArr[:])

	return marshaledElem, hashStr, err
}

func encodeAndHashElement(postgresElem interface{}) ([]byte, string, error) {
	marshaled, errMarshaled := json.Marshal(postgresElem)
	if errMarshaled != nil {
		return []byte{}, "", errMarshaled
	}

	hashArr := sha256.Sum256(marshaled)
	hashStr := hex.EncodeToString(hashArr[:])

	return marshaled, hashStr, nil
}

// CreateElement create a generic element
func CreateElement(element interface{}, name, id, elemType, status, version string, crawlTime time.Time) (*bloopi_agent.Element, error) {
	marshaled, hashed, err := encodeAndHashElement(element)
	if err != nil {
		return nil, err
	}

	if status == "" {
		status = bloopi_agent.StatusNoStatus
	}

	return &bloopi_agent.Element{
		RetrievedAt: crawlTime,
		Name:        name,
		ID:          id,
		Type:        elemType,
		Hash:        hashed,
		Data:        marshaled,
		IsJSONData:  true,
		Status:      status,
		Version:     version,
	}, nil
}

// CreateAWSElement create an AWS element
func CreateAWSElement(element interface{}, name, id, elemType, status, version string, crawlTime time.Time) (*bloopi_agent.Element, error) {
	marshaled, hashed, err := encodeAndHashAWSStruct(element)
	if err != nil {
		return nil, err
	}

	return &bloopi_agent.Element{
		RetrievedAt: crawlTime,
		Name:        name,
		ID:          id,
		Type:        elemType,
		Hash:        hashed,
		Data:        marshaled,
		IsJSONData:  false,
		Version:     version,
		Status:      status,
	}, nil
}

// CreateRelationship create a relationship element
func CreateRelationship(sourceID, destinationID, relationshipType string, relationType int, crawlTime time.Time) (*bloopi_agent.Element, error) {
	if sourceID == "" || destinationID == "" {
		return nil, errors.New("SourceID or DestinationID must both be non empty in oder to create a relationship")
	}

	parentElem := bloopi_agent.RelationshipElement{
		SourceID:         sourceID,
		DestinationID:    destinationID,
		RelationshipType: relationshipType,
		RelationType:     relationType,
	}

	relationshipWrapperElem, errRelationshipWrapperElem := CreateElement(
		parentElem,
		fmt.Sprintf("%s.%s", parentElem.SourceID, parentElem.DestinationID),
		fmt.Sprintf("%s.%s", parentElem.SourceID, parentElem.DestinationID),
		bloopi_agent.RelationshipType,
		bloopi_agent.StatusNoStatus,
		"",
		crawlTime,
	)
	if errRelationshipWrapperElem != nil {
		return nil, errRelationshipWrapperElem
	}

	return relationshipWrapperElem, nil
}

func CleanUpDataSource(inputDS *bloopi_agent.DataSource, skipFields []string) *bloopi_agent.DataSource {
	var cleanedDataSource bloopi_agent.DataSource

	cleanedDataSource.Info.Name = inputDS.Info.Name
	cleanedDataSource.Info.Type = inputDS.Info.Type
	cleanedDataSource.Info.Desc = inputDS.Info.Desc
	cleanedDataSource.DataSourceID = inputDS.DataSourceID

	for _, dsConfigKeyValue := range inputDS.Config.ValuePairs {
		if slices.Contains(skipFields, strings.ToLower(dsConfigKeyValue.Key)) {
			continue
		}

		cleanedDataSource.Config.ValuePairs = append(cleanedDataSource.Config.ValuePairs, bloopi_agent.KeyValue{
			Key:   dsConfigKeyValue.Key,
			Value: dsConfigKeyValue.Value,
		})
	}

	return &cleanedDataSource
}

func AddRelationship(existingRelationships *[]*bloopi_agent.Element, source, destination string, relationType int, crawlTime time.Time) {
	rel, errRel := CreateRelationship(source, destination, bloopi_agent.RelationshipType, relationType, crawlTime)
	if errRel == nil {
		*existingRelationships = append(*existingRelationships, rel)
	}
}
