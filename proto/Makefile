export GO111MODULE := on

.PHONY: clean
clean:
	rm -rf build
	rm -rf pkg

get-deps:
	go install github.com/bufbuild/buf/cmd/buf
	go install github.com/golang/protobuf/protoc-gen-go
	go install google.golang.org/grpc
	go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

.PHONY: build-dir
build-dir:
	mkdir -p build/proto

%.pb.go: %.proto build-dir
	protoc -I src $< --go_out=plugins=grpc:build/proto

.PHONY: proto
build: $(patsubst %.proto, %.pb.go, $(shell find src/determined -type f -name '*.proto'))
	protoc -I src src/determined/api/v1/api.proto --grpc-gateway_out=logtostderr=true:build/proto
	rm -rf pkg
	mkdir -p pkg
	cp -r build/proto/github.com/determined-ai/determined/proto/pkg/* ./pkg
	rm -rf build/proto
	mkdir -p build/swagger
	protoc -I src src/determined/api/v1/api.proto --swagger_out=logtostderr=true:build/swagger
	python3 scripts/swagger.py build/swagger

.PHONY: check
check:
	buf check lint

.PHONY: check-fmt
check-fmt:
	git ls-files -z 'src/determined/**/*.proto' | xargs -0 -P8 clang-format --dry-run --Werror

.PHONY: fmt
fmt:
	git ls-files -z 'src/determined/**/*.proto' | xargs -0 -P8 clang-format -i

