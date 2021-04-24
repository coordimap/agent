package aws

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func hash(data []byte) (string, error) {
	var v interface{}

	err := json.Unmarshal(data, &v)
	if err != nil {
		return "", err
	}

	cdoc, _ := json.Marshal(v)
	sum := sha256.Sum256(cdoc)

	return hex.EncodeToString(sum[0:]), nil
}
