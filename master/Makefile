.DEFAULT_GOAL := build
GENERATED := packaging/LICENSE
GENERATION_INPUTS = ../schemas/gen.py $(shell find ./pkg/schemas/ -name '*.go') $(shell find ../schemas/expconf -name '*.json')
MOCK_GENERATION_INPUTS = ./internal/sproto/task.go ./internal/db/database.go

export VERSION := $(shell cat ../VERSION)
export GO111MODULE := on

.PHONY: clean
clean: ungen
	rm -rf dist
	rm -rf build

.PHONY: ungen
ungen:
	rm -f $(GENERATED)
	rm -f `find ./pkg/schemas/ -name 'zgen_*.go'` build/schema_gen.stamp
	rm -f `find ./internal/mocks -name '*.go'` build/mock_gen.stamp

.PHONY: gen
gen: $(GENERATED) build/schema_gen.stamp build/mock_gen.stamp

.PHONY: force-gen
force-gen:
	rm -f build/schema_gen.stamp

build/schema_gen.stamp: $(GENERATION_INPUTS)
	go generate ./pkg/schemas/...
	mkdir -p build
	touch $@

build/mock_gen.stamp: $(MOCK_GENERATION_INPUTS)
	mockery --name=DB --dir=internal/db --output internal/mocks --filename db.go
	mockery --name=Resources --dir=internal/sproto --output internal/mocks --filename resources.go
	mockery --name=UserAuthZ --dir=internal/user --output internal/mocks --filename user_authz_iface.go
	mockery --name=WorkspaceAuthZ --dir=internal/workspace --output internal/mocks --filename workspace_authz_iface.go
	mockery --name=ProjectAuthZ --dir=internal/project --output internal/mocks --filename project_authz_iface.go
	# fmt once since the mocks are not generated formatted.
	goimports -l -local github.com/determined-ai -w ./internal/mocks
	mkdir -p build
	touch $@

.PHONY: check-gen
check-gen: force-gen gen
	# Checking that committed, generated code is up-to-date by ensuring that
	# git reports the files as unchanged after forcibly regenerating the files:
	test -z "$(shell git status --porcelain '**/zgen*')"
	test -z "$(shell git status --porcelain './internal/mocks/*')"

.PHONY: get-deps
get-deps:
	go install mvdan.cc/gofumpt@v0.3.1
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.45.2
	go install github.com/bufbuild/buf/cmd/buf@v0.42.1
	go install golang.org/x/tools/cmd/goimports@v0.1.5
	go install github.com/goreleaser/goreleaser@v1.1.0
	go install github.com/ryanbressler/CloudForest/growforest@v0.0.0-20161201194407-d014dc32840a
	go install github.com/swaggo/swag/cmd/swag@v1.7.0
	go install github.com/vektra/mockery/v2@v2.13.1

.PHONY: build
build: export DET_SEGMENT_MASTER_KEY ?=
build: export DET_SEGMENT_WEBUI_KEY ?=
build: gen
	go build \
		-ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentMasterKey=$(DET_SEGMENT_MASTER_KEY) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentWebUIKey=$(DET_SEGMENT_WEBUI_KEY)" \
		-o build/determined-master \
		./cmd/determined-master

.PHONY: build-race
build-race: export DET_SEGMENT_MASTER_KEY ?=
build-race: export DET_SEGMENT_WEBUI_KEY ?=
build-race: gen
	go build \
		-ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentMasterKey=$(DET_SEGMENT_MASTER_KEY) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentWebUIKey=$(DET_SEGMENT_WEBUI_KEY)" \
		-o build/determined-master \
		-race \
		./cmd/determined-master

.PHONY: check
check: check-gen
	go mod tidy
	git diff --quiet go.mod go.sum
	golangci-lint run -v --timeout 10m

.PHONY: fmt
fmt:
	goimports -l -local github.com/determined-ai -w .
	gofumpt -l -w .

.PHONY: test
test:
	go test -short -coverprofile=coverage.out -covermode count -cover ./...

.PHONY: test-intg
test-intg: export DET_INTEGRATION_POSTGRES_URL ?= postgres://postgres:postgres@localhost:5432/determined?sslmode=disable
test-intg: export DET_INTEGRATION_ES_HOST ?= localhost
test-intg: export DET_INTEGRATION_ES_PORT ?= 9200
test-intg:
	go test -tags=integration -coverprofile=coverage.out -covermode count -cover ./...

.PHONY: pre-package
pre-package:
	rm -rf build
	mkdir -p build/webui/docs
	mkdir -p build/webui/react
	mkdir -p build/wheels/
	cp -r ../proto/build/swagger build/swagger
	cp -r ../docs/site/html/* build/webui/docs
	cp -r ../webui/react/build/* build/webui/react
	cp ../harness/dist/*.whl build/wheels/
	cp $(shell packaging/fetch-cloud-forest.sh) build/

.PHONY: package
package: export DET_SEGMENT_MASTER_KEY ?=
package: export DET_SEGMENT_WEBUI_KEY ?=
package: export GORELEASER_CURRENT_TAG := $(VERSION)
package: gen
	goreleaser --snapshot --rm-dist

.PHONY: release
release: export DET_SEGMENT_MASTER_KEY ?=
release: export DET_SEGMENT_WEBUI_KEY ?=
release: export GORELEASER_CURRENT_TAG := $(VERSION)
release: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -v "rc\|v" | grep "$(VERSION)" -A1 | sed -n '2 p')
release: gen
	goreleaser --rm-dist

.PHONY: publish
publish:
	docker push determinedai/determined-master:$(VERSION)

.PHONY: publish-dev
publish-dev:
	docker push determinedai/determined-master:$(shell git rev-parse HEAD)
	docker push determinedai/determined-dev:determined-master-$(shell git rev-parse HEAD)

packaging/LICENSE: $(shell find ../tools/scripts/licenses -type f)
	../tools/scripts/gen-attributions.py master $@
