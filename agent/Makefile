.DEFAULT_GOAL := build
SHELL := bash

export VERSION:=$(shell cat ../VERSION)
export GO111MODULE := on
export DOCKER_REPO ?= determinedai

FULL_COMMIT = $(shell git rev-parse HEAD)
SHORT_COMMIT = $(shell git rev-parse HEAD | head -c9)
PROJECT_NAME = determined-agent
ARCHS = amd64 arm64 ppc64
MULTI_ARCH_IMAGES = $(shell for arch in $(ARCHS); do echo $(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT)-$$arch; done)

NVCR_REPO ?= nvcr.io/isv-ngc-partner/determined

PUB_MANIFESTS = \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT) \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(SHORT_COMMIT) \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(VERSION)

DEV_MANIFESTS = \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT) \
	$(DOCKER_REPO)/determined-dev:$(PROJECT_NAME)-$(FULL_COMMIT)

NVCR_TAGS = \
	$(NVCR_REPO)/$(PROJECT_NAME):$(FULL_COMMIT) \
	$(NVCR_REPO)/$(PROJECT_NAME):$(SHORT_COMMIT) \
	$(NVCR_REPO)/$(PROJECT_NAME):$(VERSION)

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
	gotestsum --junitfile test.junit.xml -- -race -coverprofile=coverage.out -covermode atomic ./...

.PHONY: test-intg
test-intg: export DET_INTEGRATION_POSTGRES_URL ?= postgres://postgres:postgres@localhost:5432/determined?sslmode=disable
test-intg: export DET_INTEGRATION_ES_HOST ?= localhost
test-intg: export DET_INTEGRATION_ES_PORT ?= 9200
test-intg:
	gotestsum --junitfile test-intg.junit.xml -- -p 1 -race -tags=integration -coverprofile=coverage.out -covermode atomic ./...

.PHONY: buildx
buildx:
	docker context rm -f buildx-context || true
	docker context create buildx-context
	docker buildx rm -f buildx-build || true
	{ \
		platforms=(); \
		for arch in $(ARCHS); do \
			[ $$arch == "ppc64" ] && arch="ppc64le"; \
			platforms+=("linux/$$arch"); \
		done; \
		platform_list=$$(IFS=, ; echo "$${platforms[*]}"); \
		docker --context=buildx-context buildx create --platform $$platform_list --bootstrap --use --name buildx-build; \
	}
	docker buildx ls

packaging/LICENSE: $(shell find ../tools/scripts/licenses -type f)
	../tools/scripts/gen-attributions.py agent $@

.PHONY: package
package: export GORELEASER_CURRENT_TAG := $(VERSION)
package: packaging/LICENSE buildx
	goreleaser --snapshot --rm-dist

.PHONY: release
release: export GORELEASER_CURRENT_TAG := $(VERSION)
release: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^[0-9.]+$$' | grep "$(VERSION)" -A1 | sed -n '2 p')
release: packaging/LICENSE buildx
	goreleaser --rm-dist
	make publish-nvcr

define manifest_publish
	for image in $(2); do \
		docker push $$image; \
	done
	for manifest in $(1); do \
		docker manifest rm $$manifest; \
		docker manifest create $$manifest $(2); \
		docker manifest push $$manifest || exit 1; \
	done
endef

.PHONY: publish-dev
publish-dev:
	@$(call manifest_publish, $(DEV_MANIFESTS), $(MULTI_ARCH_IMAGES))

.PHONY: publish
publish:
	@$(call manifest_publish, $(PUB_MANIFESTS), $(MULTI_ARCH_IMAGES))

.PHONY: publish-nvcr
publish-nvcr:
	for image in $(NVCR_TAGS); do \
		docker tag $(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT)-amd64 $$image; \
		docker push $$image; \
	done
