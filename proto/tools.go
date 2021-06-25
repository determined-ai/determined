// +build tools

package tools

import (
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger"
	_ "github.com/swaggo/swag/cmd/swag"
	_ "google.golang.org/grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
