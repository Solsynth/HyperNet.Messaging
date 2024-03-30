package models

import jsoniter "github.com/json-iterator/go"

func FitStruct(src any, out any) {
	raw, _ := jsoniter.Marshal(src)
	_ = jsoniter.Unmarshal(raw, out)
}
