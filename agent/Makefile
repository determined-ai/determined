export VERSION:=$(shell cat ../VERSION)
export GO111MODULE := on
.DEFAULT_GOAL := build

.PHONY: clean
clean:
	rm -rf dist
	rm -rf build

.PHONY: get-deps
get-deps:
	./get-deps.sh

.PHONY: build
build:
	go build \
		-ldflags "-X main.version=$(VERSION)" \
		-o build/determined-agent \
		./cmd/determined-agent

.PHONY: build-race
build-race:
	go build \
		-ldflags "-X main.version=$(VERSION)" \
		-o build/determined-agent \
		-race \
		./cmd/determined-agent

.PHONY: check
check:
	go mod tidy
	git diff --quiet go.mod go.sum
	golangci-lint --build-tags integration run -v --timeout 10m

.PHONY: fmt
fmt:
	goimports -l -local github.com/determined-ai -w .
	gofumpt -l -w .

.PHONY: test
test:
	go test -coverprofile=coverage.out -covermode count ./...

.PHONY: test-intg
test-intg: export DET_INTEGRATION_POSTGRES_URL ?= postgres://postgres:postgres@localhost:5432/determined?sslmode=disable
test-intg: export DET_INTEGRATION_ES_HOST ?= localhost
test-intg: export DET_INTEGRATION_ES_PORT ?= 9200
test-intg:
	go test -tags=integration -coverprofile=coverage.out -covermode count ./...

packaging/LICENSE: $(shell find ../tools/scripts/licenses -type f)
	../tools/scripts/gen-attributions.py agent $@

.PHONY: package
package: export GORELEASER_CURRENT_TAG := $(VERSION)
package: packaging/LICENSE
	goreleaser --snapshot --rm-dist

.PHONY: release
release: export GORELEASER_CURRENT_TAG := $(VERSION)
release: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^[0-9.]+$$' | grep "$(VERSION)" -A1 | sed -n '2 p')
release: packaging/LICENSE
	goreleaser --rm-dist

.PHONY: publish
publish:
	docker push determinedai/determined-agent:$(VERSION)

.PHONY: publish-dev
publish-dev:
	docker push determinedai/determined-agent:$(shell git rev-parse HEAD)
	docker push determinedai/determined-dev:determined-agent-$(shell git rev-parse HEAD)
