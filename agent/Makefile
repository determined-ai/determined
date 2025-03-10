.DEFAULT_GOAL := build
SHELL := bash

export VERSION:=$(shell ../version.sh)
export VERSION_TAG:=$(shell ../version.sh -t)
export VERSION_DOCKER := $(shell ../version.sh -d)

export GO111MODULE := on
export DOCKER_REPO ?= determinedai

FULL_COMMIT = $(shell git rev-parse HEAD)
SHORT_COMMIT = $(shell git rev-parse HEAD | head -c9)
PROJECT_NAME = determined-agent
PROJECT_NAME_DRYRUN = determined-agent-dryrun
EE_PROJECT_NAME = hpe-mlde-agent
ARCHS = amd64 arm64
MULTI_ARCH_IMAGES = $(shell for arch in $(ARCHS); do echo $(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT)-$$arch; done)
MULTI_ARCH_IMAGES_DRYRUN = $(shell for arch in $(ARCHS); do echo $(DOCKER_REPO)$(PROJECT_NAME_DRYRUN):$(FULL_COMMIT)-$$arch; done)
EE_MULTI_ARCH_IMAGES = $(shell for arch in $(ARCHS); do echo $(DOCKER_REPO)/$(EE_PROJECT_NAME):$(FULL_COMMIT)-$$arch; done)

NVCR_REPO ?= nvcr.io/isv-ngc-partner/determined

PUB_MANIFESTS = \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT) \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(SHORT_COMMIT) \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(VERSION)

PUB_MANIFESTS_DRYRUN = \
	$(DOCKER_REPO)/$(PROJECT_NAME_DRYRUN):$(FULL_COMMIT) \
	$(DOCKER_REPO)/$(PROJECT_NAME_DRYRUN):$(SHORT_COMMIT) \
	$(DOCKER_REPO)/$(PROJECT_NAME_DRYRUN):$(VERSION)

EE_PUB_MANIFESTS = \
	$(DOCKER_REPO)/$(EE_PROJECT_NAME):$(FULL_COMMIT) \
	$(DOCKER_REPO)/$(EE_PROJECT_NAME):$(SHORT_COMMIT) \
	$(DOCKER_REPO)/$(EE_PROJECT_NAME):$(VERSION)

DEV_MANIFESTS = \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT) \
	$(DOCKER_REPO)/determined-dev:$(PROJECT_NAME)-$(FULL_COMMIT)

EE_DEV_MANIFESTS = \
	$(DOCKER_REPO)/$(EE_PROJECT_NAME):$(FULL_COMMIT)

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
	CGO_ENABLED=0 go build \
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
	git diff --quiet ../go.mod ../go.sum
	golangci-lint --config ../master/.golangci.yml --build-tags integration run -v --timeout 10m

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
	gotestsum --junitfile test-intg.junit.xml -- -race -tags=integration -coverprofile=coverage.out -covermode atomic ./...

.PHONY: buildx
buildx:
	docker context rm -f buildx-context || true
	docker context create buildx-context
	docker buildx rm -f buildx-build || true
	{ \
		platforms=(); \
		for arch in $(ARCHS); do \
			platforms+=("linux/$$arch"); \
		done; \
		platform_list=$$(IFS=, ; echo "$${platforms[*]}"); \
		docker --context=buildx-context buildx create --platform $$platform_list --bootstrap --use --name buildx-build; \
	}
	docker buildx ls

packaging/LICENSE: $(shell find ../tools/scripts/licenses -type f)
	../tools/scripts/gen-attributions.py agent $@

.PHONY: package
package: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
package: packaging/LICENSE buildx
	goreleaser --snapshot --rm-dist

.PHONY: package-dryrun
package-dryrun: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
package-dryrun: packaging/LICENSE buildx
	goreleaser --snapshot --rm-dist -f ./.goreleaser_dryrun.yml

.PHONY: package-ee
package-ee: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)-ee
package-ee: packaging/LICENSE buildx
	goreleaser --snapshot --rm-dist -f ./.goreleaser_ee.yml

.PHONY: release
release: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
release: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^v?[0-9.]+$$' | grep "$(VERSION_TAG)" -A1 | sed -n '2 p')
release: packaging/LICENSE buildx
	goreleaser --rm-dist
	make publish-nvcr

.PHONY: release-dryrun
release-dryrun: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
release-dryrun: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^v?[0-9.]+$$' | grep "$(VERSION_TAG)" -A1 | sed -n '2 p')
# We intentionally do not invoke `make publish-nvcr(-dryrun)` here.
release-dryrun: packaging/LICENSE buildx
	goreleaser --rm-dist -f ./.goreleaser_dryrun.yml

.PHONY: release-ee
release-ee: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)-ee
release-ee: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^v?[0-9.]+-ee$$' | grep "$(GORELEASER_CURRENT_TAG)" -A1 | sed -n '2 p')
release-ee: packaging/LICENSE buildx
	goreleaser --rm-dist -f ./.goreleaser_ee.yml

.PHONY: release-ee-dryrun
# VERSION_TAG v0.38.0+dryrun becomes v0.38.0-ee+dryrun
release-ee-dryrun: export GORELEASER_CURRENT_TAG := $(shell echo $(VERSION_TAG) | sed "s/+dryrun/-ee&/g")
release-ee-dryrun: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^v?[0-9.]+-ee$$' | grep "$(GORELEASER_CURRENT_TAG)" -A1 | sed -n '2 p')
release-ee-dryrun: packaging/LICENSE buildx
	goreleaser --rm-dist -f ./.goreleaser_ee_dryrun.yml

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

.PHONY: publish-dev-ee
publish-dev-ee:
	@$(call manifest_publish, $(EE_DEV_MANIFESTS), $(EE_MULTI_ARCH_IMAGES))

.PHONY: publish
publish:
	@$(call manifest_publish, $(PUB_MANIFESTS), $(MULTI_ARCH_IMAGES))

.PHONY: publish-dryrun
publish-dryrun:
	@$(call manifest_publish, $(PUB_MANIFESTS_DRYRUN), $(MULTI_ARCH_IMAGES_DRYRUN))

.PHONY: publish-ee
publish-ee:
	@$(call manifest_publish, $(EE_PUB_MANIFESTS), $(EE_MULTI_ARCH_IMAGES))

.PHONY: publish-nvcr
publish-nvcr:
	for image in $(NVCR_TAGS); do \
		docker tag $(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT)-amd64 $$image; \
		docker push $$image; \
	done
