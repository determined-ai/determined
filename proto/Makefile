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

go_src_path := ../master/internal
echo_swagger_source_files := $(shell find $(go_src_path) -type f -name 'core*.go')
echo_swagger_patch_dir := build/echo-swagger
echo_swagger_patch := $(echo_swagger_patch_dir)/swagger.json

# Buf image binary
buf_image := buf.image.bin

.PHONY: build
build: build/proto.stamp $(swagger_out)

.PHONY: clean
clean:
	rm -rf build pkg

.PHONY: get-deps
get-deps:
	./get-deps.sh

build/proto.stamp: $(source_files)
	protoc --version |perl -ne '/.*(\d+)\.(\d+)\.(\d+)/&&(($$1==3&&$$2<15)||$$1<3)&&print("\nERROR: \
		Detected protoc version: $$1.$$2.$$3. Please upgrade protoc to version 3.15.0 or above\n\n.")&&exit 1;'
	rm -rf build/proto pkg
	mkdir -p build/proto
	# Protobuf generation.
	for source in $(source_files) ; do protoc -I src "$$source" --go_out=plugins=grpc:build/proto || exit 1; done
	# GRPC generation.
	protoc -I src $(grpc_in) --grpc-gateway_out=logtostderr=true:build/proto --grpc-gateway_opt allow_delete_body=true
	mv $(build_path) pkg
ifeq "$(shell uname)" "Darwin"
	sed -E -i '' '/\/\/[[:space:]]*protoc/d' pkg/*/*.pb.go
else
	sed -E -i '/\/\/[[:space:]]*protoc/d' pkg/*/*.pb.go
endif
	touch $@

$(echo_swagger_patch_dir):
	mkdir -p $(echo_swagger_patch_dir)

$(echo_swagger_patch): $(echo_swagger_patch_dir) $(echo_swagger_source_files)
	swag init -g ../master/cmd/determined-master/main.go -d ../master/. -o $(echo_swagger_patch_dir)
	jq 'del(.swagger, .info)' $(echo_swagger_patch) > $(echo_swagger_patch).tmp
	mv $(echo_swagger_patch).tmp $(echo_swagger_patch)

build/swagger:
	mkdir -p build/swagger

$(swagger_out): $(source_files) build/swagger $(swagger_patch) $(echo_swagger_patch)
	protoc -I src $(swagger_in) --swagger_out=logtostderr=true,allow_delete_body=true,json_names_for_fields=true:build/swagger
	python3 scripts/swagger.py $@ $(swagger_patch)
	python3 scripts/swagger.py $@ $(echo_swagger_patch)

# Update buf image for breaking change check.
.PHONY: gen-buf-image
gen-buf-image:
	buf build -o $(buf_image)

.PHONY: check
check: check-fmt build/proto.stamp
	buf lint
	buf breaking --against $(buf_image)
	go mod tidy
	git diff go.mod go.sum pkg/
	git diff --quiet go.mod go.sum pkg/

.PHONY: check-fmt
check-fmt:
	git ls-files -z 'src/determined/**/*.proto' | xargs -0 -P8 clang-format --dry-run --Werror

.PHONY: fmt
fmt:
	git ls-files -z 'src/determined/**/*.proto' | xargs -0 -P8 clang-format -i
