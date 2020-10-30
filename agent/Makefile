export VERSION:=$(shell cat ../VERSION)
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
		-ldflags "-X main.version=$(VERSION)" \
		-o build/determined-agent \
		./cmd/determined-agent

.PHONY: check
check:
	golangci-lint run

.PHONY: fmt
fmt:
	goimports -l -local github.com/determined-ai -w .

.PHONY: test
test:
	go test ./...

packaging/LICENSE: $(shell find ../tools/scripts/licenses -type f)
	../tools/scripts/gen-attributions.py agent $@

.PHONY: package
package: export GORELEASER_CURRENT_TAG := $(VERSION)
package: packaging/LICENSE
	goreleaser --snapshot --rm-dist

.PHONY: release
release: export GORELEASER_CURRENT_TAG := $(VERSION)
release: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -v "rc" | grep "v$(VERSION)" -A1 | sed -n '2 p')
release: packaging/LICENSE
	goreleaser --rm-dist

.PHONY: publish
publish:
	docker push determinedai/determined-agent:$(VERSION)

.PHONY: publish-dev
publish-dev:
	docker push determinedai/determined-agent:$(shell git rev-parse HEAD)
	docker push determinedai/determined-dev:determined-agent-$(shell git rev-parse HEAD)
