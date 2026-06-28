package object

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Encode 将对象序列化为 JSON 字节，作为持久化和 CID 计算的统一编码格式。
func Encode(obj Object) ([]byte, error) {
	return json.Marshal(obj)
}

// Decode 将 JSON 字节反序列化为对象。
func Decode(data []byte) (Object, error) {
	var obj Object
	if err := json.Unmarshal(data, &obj); err != nil {
		return Object{}, err
	}
	return obj, nil
}

// CID 基于对象的规范 JSON 编码计算 SHA-256 内容标识。
func CID(obj Object) (string, error) {
	data, err := Encode(obj)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
