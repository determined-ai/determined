export VERSION:=$(shell cat ../VERSION)
export GO111MODULE := on

.PHONY: clean
clean:
	rm -rf dist
	rm -rf coverage.out

.PHONY: get-deps
get-deps:
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go install golang.org/x/tools/cmd/goimports
	go install github.com/goreleaser/goreleaser

.PHONY: build
build:
	go build \
		-ldflags "-X github.com/determined-ai/determined/agent/version.Version=$(VERSION)" \
		-o build/determined-agent \
		./cmd/determined-agent

.PHONY: install
install:
	go install -ldflags "-X github.com/determined-ai/determined/agent/version.Version=$(VERSION)" ./cmd/determined-agent

.PHONY: check
check:
	golangci-lint run

.PHONY: fmt
fmt:
	goimports -l -local github.com/determined-ai -w .

.PHONY: test
test:
	go test ./...

.PHONY: package
package:
	GORELEASER_CURRENT_TAG=$(VERSION) goreleaser --snapshot --rm-dist

.PHONY: publish
publish:
	docker push determinedai/determined-agent:$(VERSION)

.PHONY: publish-dev
publish-dev:
	docker push determinedai/determined-agent:$(shell git rev-parse HEAD)
	docker push determinedai/determined-dev:determined-agent-$(shell git rev-parse HEAD)
