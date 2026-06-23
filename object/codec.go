package object

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// 序列化
func Encode(obj Object) ([]byte, error) {
	return json.Marshal(obj)
}

// 反序列化
func Decode(data []byte) (Object, error) {
	var obj Object
	if err := json.Unmarshal(data, &obj); err != nil {
		return Object{}, err
	}
	return obj, nil
}

// 计算对象的CID
func CID(obj Object) (string, error) {
	data, err := Encode(obj)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
