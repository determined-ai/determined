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
	@rm -rf build/root
	@mkdir -p build/root/webui/docs
	@mkdir -p build/root/webui/elm
	@mkdir -p build/root/webui/react
	@mkdir -p build/root/wheels/
	cp -r static/ build/root/
	cp -r ../docs/site/html/* build/root/webui/docs
	cp -r ../webui/react/build/* build/root/webui/react
	cp -r ../webui/elm/public build/root/webui/elm
	cp -r ../common/dist/*.whl build/root/wheels/
	cp -r ../cli/dist/*.whl build/root/wheels/
	cp -r ../harness/dist/*.whl build/root/wheels/
	go build \
		-ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION)" \
		-o build/determined-master \
		./cmd/determined-master

.PHONY: check
check:
	golangci-lint run

.PHONY: fmt
fmt:
	goimports -l -local github.com/determined-ai -w .

.PHONY: test
test:
	go test -v -short -coverprofile=coverage.out -covermode count -cover ./...

.PHONY: package
package: export DET_SEGMENT_MASTER_KEY ?= ""
package: export DET_SEGMENT_WEBUI_KEY ?= ""
package: export GORELEASER_CURRENT_TAG := $(VERSION)
package:
	goreleaser --snapshot --rm-dist

.PHONY: release
release: export DET_SEGMENT_MASTER_KEY ?= ""
release: export DET_SEGMENT_WEBUI_KEY ?= ""
release: export GORELEASER_CURRENT_TAG := $(VERSION)
release: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -v "rc" | grep "v$(VERSION)" -A1 | sed -n '2 p')
release:
	goreleaser --rm-dist

.PHONY: publish
publish:
	docker push determinedai/determined-master:$(VERSION)

.PHONY: publish-dev
publish-dev:
	docker push determinedai/determined-master:$(shell git rev-parse HEAD)
	docker push determinedai/determined-dev:determined-master-$(shell git rev-parse HEAD)
