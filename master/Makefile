export VERSION := $(shell cat ../VERSION)
export GO111MODULE := on

.PHONY: clean
clean:
	rm -rf dist
	rm -rf build

.PHONY: get-deps
get-deps:
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go install golang.org/x/tools/cmd/goimports
	go install github.com/goreleaser/goreleaser

.PHONY: build
build:
	go build \
		-ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION)" \
		-o build/determined-master \
		./cmd/determined-master

.PHONY: check
check:
	golangci-lint run --timeout 10m

.PHONY: fmt
fmt:
	goimports -l -local github.com/determined-ai -w .

.PHONY: test
test:
	go test -short -coverprofile=coverage.out -covermode count -cover ./...

.PHONY: test
test-intg: export DET_INTEGRATION_POSTGRES_URL ?= postgres://postgres:postgres@localhost:5432/determined?sslmode=disable
test-intg:
	go test -tags=integration -coverprofile=coverage.out -covermode count -cover ./test/...

.PHONY: pre-package
pre-package:
	rm -rf build
	mkdir -p build/webui/docs
	mkdir -p build/webui/react
	mkdir -p build/wheels/
	cp -r ../proto/build/swagger build/swagger
	cp -r ../docs/site/html/* build/webui/docs
	cp -r ../webui/react/build/* build/webui/react
	cp ../common/dist/*.whl build/wheels/
	cp ../cli/dist/*.whl build/wheels/
	cp ../harness/dist/*.whl build/wheels/

packaging/LICENSE: $(shell find ../tools/scripts/licenses -type f)
	../tools/scripts/gen-attributions.py agent $@

.PHONY: package
package: export DET_SEGMENT_MASTER_KEY ?=
package: export DET_SEGMENT_WEBUI_KEY ?=
package: export GORELEASER_CURRENT_TAG := $(VERSION)
package: packaging/LICENSE
	goreleaser --snapshot --rm-dist

.PHONY: release
release: export DET_SEGMENT_MASTER_KEY ?=
release: export DET_SEGMENT_WEBUI_KEY ?=
release: export GORELEASER_CURRENT_TAG := $(VERSION)
release: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -v "rc\|v" | grep "$(VERSION)" -A1 | sed -n '2 p')
release: packaging/LICENSE
	goreleaser --rm-dist

.PHONY: publish
publish:
	docker push determinedai/determined-master:$(VERSION)

.PHONY: publish-dev
publish-dev:
	docker push determinedai/determined-master:$(shell git rev-parse HEAD)
	docker push determinedai/determined-dev:determined-master-$(shell git rev-parse HEAD)
