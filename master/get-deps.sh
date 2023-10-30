# Proto deps.
go install github.com/bufbuild/buf/cmd/buf@v0.42.1
go install github.com/golang/protobuf/protoc-gen-go@v1.5.2
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0
go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.14.6
go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@v1.14.6

# Master deps.
go install mvdan.cc/gofumpt@v0.4.0
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.1
go install github.com/bufbuild/buf/cmd/buf@v0.42.1
go install golang.org/x/tools/cmd/goimports@v0.1.5
go install github.com/goreleaser/goreleaser@v1.14.1
go install github.com/swaggo/swag/cmd/swag@v1.8.9
go install github.com/vektra/mockery/v2@v2.20.0
go install gotest.tools/gotestsum@v1.9.0
