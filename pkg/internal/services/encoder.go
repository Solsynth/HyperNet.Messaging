package services

import jsoniter "github.com/json-iterator/go"

func EncodeJSONBody(in any) []byte {
	out, _ := jsoniter.Marshal(in)
	return out
}
