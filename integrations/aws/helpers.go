package aws

import (
	"bytes"
	"encoding/gob"
)

func encodeStruct(elem interface{}) ([]byte, error) {
	var buff bytes.Buffer
	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(elem)
	marshaledElem := buff.Bytes()
	buff.Reset()

	return marshaledElem, err
}
