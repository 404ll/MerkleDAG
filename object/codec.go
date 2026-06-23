package object

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func Encode(obj Object) ([]byte, error) {
	return json.Marshal(obj)
}

func Decode(data []byte) (Object, error) {
	var obj Object
	if err := json.Unmarshal(data, &obj); err != nil {
		return Object{}, err
	}
	return obj, nil
}

func CID(obj Object) (string, error) {
	data, err := Encode(obj)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
