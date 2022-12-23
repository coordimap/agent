package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
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
	marshaledElem := []byte(base64.StdEncoding.EncodeToString(buff.Bytes()))
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

func CreateElement(element interface{}, name, id, elemType string, crawlTime time.Time) (*bloopi_agent.Element, error) {
	marshaled, hashed, err := encodeAndHashElement(element)
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
		IsJSONData:  true,
	}, nil
}

func CreateAWSElement(element interface{}, name, id, elemType string, crawlTime time.Time) (*bloopi_agent.Element, error) {
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
	}, nil
}
