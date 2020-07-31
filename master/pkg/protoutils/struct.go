package protoutils

import (
	"encoding/json"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/protobuf/encoding/protojson"
)

// ToStruct converts a Go interface to a protobuf struct.
func ToStruct(v interface{}) *structpb.Struct {
	b, _ := json.Marshal(v)
	configStruct := &structpb.Struct{}
	_ = protojson.Unmarshal(b, configStruct)
	return configStruct
}
