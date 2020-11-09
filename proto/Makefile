export GO111MODULE := on

# String constants.
build_path := build/proto/github.com/determined-ai/determined/proto/pkg
src_path := src/determined

# Pre-defined list of all source files.
source_files := $(shell find $(src_path) -type f -name '*.proto')

grpc_in := $(src_path)/api/v1/api.proto

# Currently only one file needs Swagger treatment.
swagger_in := $(src_path)/api/v1/api.proto
swagger_out := build/swagger/determined/api/v1/api.swagger.json
swagger_patch := patches/api.json

# Buf image binary
buf_image := buf.image.bin

.PHONY: build
build: build/proto.stamp $(swagger_out)

.PHONY: clean
clean:
	rm -rf build pkg

get-deps:
	go install github.com/bufbuild/buf/cmd/buf
	go install github.com/golang/protobuf/protoc-gen-go
	go install google.golang.org/grpc
	go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

build/proto.stamp: $(source_files)
	rm -rf build/proto pkg
	mkdir -p build/proto
	# Protobuf generation.
	for source in $(source_files) ; do protoc -I src "$$source" --go_out=plugins=grpc:build/proto ; done
	# GRPC generation.
	protoc -I src $(grpc_in) --grpc-gateway_out=logtostderr=true:build/proto
	mv $(build_path) pkg
	touch $@

build/swagger:
	mkdir -p build/swagger

$(swagger_out): $(source_files) build/swagger
	protoc -I src $(swagger_in) --swagger_out=logtostderr=true,json_names_for_fields=true:build/swagger
	python3 scripts/swagger.py $@ $(swagger_patch)

# Update buf image for breaking change check.
.PHONY: gen-buf-image
gen-buf-image: check
	# disallow untracked or dirty files.
	test -z "$(shell git status --porcelain)"
	buf image build -o $(buf_image)

.PHONY: check
check:
	buf check lint
	buf check breaking --against-input $(buf_image)

.PHONY: check-fmt
check-fmt:
	git ls-files -z 'src/determined/**/*.proto' | xargs -0 -P8 clang-format --dry-run --Werror

.PHONY: fmt
fmt:
	git ls-files -z 'src/determined/**/*.proto' | xargs -0 -P8 clang-format -i

