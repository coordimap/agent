package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
)

func encodeAndHashElement(postgresElem interface{}) ([]byte, string, error) {
	marshaled, errMarshaled := json.Marshal(postgresElem)
	if errMarshaled != nil {
		return []byte{}, "", errMarshaled
	}

	hashArr := sha256.Sum256(marshaled)
	hashStr := hex.EncodeToString(hashArr[:])

	return marshaled, hashStr, nil
}

func createElement(element interface{}, name, id, elemType string) (*bloopi_agent.Element, error) {
	marshaled, hashed, err := encodeAndHashElement(element)
	if err != nil {
		return nil, err
	}

	return &bloopi_agent.Element{
		RetrievedAt: time.Now().UTC(),
		Name:        name,
		ID:          id,
		Type:        elemType,
		Hash:        hashed,
		Data:        marshaled,
	}, nil
}
