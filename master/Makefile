.DEFAULT_GOAL := build
GENERATED := packaging/LICENSE
GENERATION_INPUTS = ../schemas/gen.py $(shell find ./pkg/schemas/ -name '*.go') $(shell find ../schemas/expconf -name '*.json')

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

.PHONY: gen-mocks
gen-mocks:
	mockery --name=DB --dir=internal/db --output internal/mocks --filename db.go
	mockery --name=Allocation --dir=internal/sproto --output internal/mocks --filename allocation.go

.PHONY: gen
gen: $(GENERATED) build/schema_gen.stamp gen-mocks

.PHONY: force-gen
force-gen:
	rm -f build/schema_gen.stamp

build/schema_gen.stamp: $(GENERATION_INPUTS)
	go generate ./pkg/schemas/...
	mkdir -p build
	touch $@

.PHONY: check-gen
check-gen: force-gen gen
	# Checking that committed, generated code is up-to-date by ensuring that
	# git reports the files as unchanged after forcibly regenerating the files:
	test -z "$(shell git status --porcelain '**/zgen*')"

.PHONY: get-deps
get-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go install golang.org/x/tools/cmd/goimports
	go install github.com/goreleaser/goreleaser
	# For hyperparameter importance computation
	go get github.com/ryanbressler/CloudForest
	go install github.com/ryanbressler/CloudForest/growforest
	go get github.com/vektra/mockery/v2/.../

.PHONY: build
build: gen
	go build \
		-ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION)" \
		-o build/determined-master \
		./cmd/determined-master

.PHONY: build-race
build-race: gen
	go build \
		-ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION)" \
		-o build/determined-master \
		-race \
		./cmd/determined-master

.PHONY: check
check: check-gen
	go mod tidy
	git diff --quiet go.mod go.sum
	golangci-lint run --timeout 10m

.PHONY: fmt
fmt:
	goimports -l -local github.com/determined-ai -w .

.PHONY: test
test:
	go test -short -coverprofile=coverage.out -covermode count -cover ./...

.PHONY: test-intg
test-intg: export DET_INTEGRATION_POSTGRES_URL ?= postgres://postgres:postgres@localhost:5432/determined?sslmode=disable
test-intg: export DET_INTEGRATION_ES_HOST ?= localhost
test-intg: export DET_INTEGRATION_ES_PORT ?= 9200
test-intg:
	go test -v -tags=integration -coverprofile=coverage.out -covermode count -cover ./test/...

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
