.DEFAULT_GOAL := build
SHELL := bash
LICENSE := packaging/LICENSE
SCHEMA_INPUTS = ../schemas/gen.py $(shell find ./pkg/schemas/ -name 'zgen_*.go' -prune -o -name '*.go' -print) $(shell find ../schemas/expconf -name '*.json')
STREAM_INPUTS = $(shell find ./internal/stream/ -name '*_test.go' -prune -o -name '*.go' -print)
STREAM_PYTHON_CLIENT = ../harness/determined/common/streams/wire.py
STREAM_TS_CLIENT = ../webui/react/src/services/stream/wire.ts
MOCK_INPUTS = Makefile ./internal/sproto/task.go ./internal/db/database.go ./internal/command/authz_iface.go ../go.mod ../go.sum ./internal/rm/resource_manager_iface.go ./internal/task/allocation_service_iface.go
GORELEASER = goreleaser

export VERSION := $(shell ../version.sh)
export VERSION_TAG :=$(shell ../version.sh -t)
export VERSION_DOCKER := $(shell ../version.sh -d)

export GO111MODULE := on

# The Docker Hub organization.
export DOCKER_REPO ?= determinedai

FULL_COMMIT = $(shell git rev-parse HEAD)
SHORT_COMMIT = $(shell git rev-parse HEAD | head -c9)
PROJECT_NAME = determined-master
PROJECT_NAME_DRYRUN = determined-master-dryrun
EE_PROJECT_NAME = hpe-mlde-master
ARCHS = amd64 arm64
ARCH_SMALL = amd64-shared-cluster
MULTI_ARCH_IMAGES = $(shell for arch in $(ARCHS); do echo $(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT)-$$arch; done)
MULTI_ARCH_IMAGES_DRYRUN = $(shell for arch in $(ARCHS); do echo $(DOCKER_REPO)/$(PROJECT_NAME_DRYRUN):$(FULL_COMMIT)-$$arch; done)
EE_MULTI_ARCH_IMAGES = $(shell for arch in $(ARCHS); do echo $(DOCKER_REPO)/$(EE_PROJECT_NAME):$(FULL_COMMIT)-$$arch; done)
SHARED_CLUSTER_IMAGE=$(shell echo $(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT)-$(ARCH_SMALL))

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

DEV_MANIFESTS_SMALL = \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT)-shared-cluster \
	$(DOCKER_REPO)/determined-dev:$(PROJECT_NAME)-$(FULL_COMMIT)-shared-cluster

NVCR_TAGS = \
	$(NVCR_REPO)/$(PROJECT_NAME):$(FULL_COMMIT) \
	$(NVCR_REPO)/$(PROJECT_NAME):$(SHORT_COMMIT) \
	$(NVCR_REPO)/$(PROJECT_NAME):$(VERSION)

.PHONY: clean
clean: ungen
	rm -rf dist
	rm -rf build

.PHONY: ungen
ungen:
	rm -f $(LICENSE)
	rm -f `find ./pkg/schemas/ -name 'zgen_*.go'` build/schema_gen.stamp
	rm -f `find ./internal/mocks -name '*.go'` build/mock_gen.stamp

.PHONY: gen
gen: $(LICENSE) build/schema_gen.stamp $(STREAM_PYTHON_CLIENT) $(STREAM_TS_CLIENT)

.PHONY: force-gen
force-gen:
	rm -f build/schema_gen.stamp
	rm -f $(STREAM_PYTHON_CLIENT)
	rm -f $(STREAM_TS_CLIENT)

build/schema_gen.stamp: $(SCHEMA_INPUTS)
	go generate ./pkg/schemas/...
	mkdir -p build
	touch $@

$(STREAM_PYTHON_CLIENT): build/stream-gen $(STREAM_INPUTS)
	build/stream-gen $(STREAM_INPUTS) --python --output $@

$(STREAM_TS_CLIENT): build/stream-gen $(STREAM_INPUTS)
	build/stream-gen $(STREAM_INPUTS) --ts --output $@

.PHONY: stream-gen
stream-gen: $(STREAM_PYTHON_CLIENT) $(STREAM_TS_CLIENT)

.PHONY: mocks
mocks: build/mock_gen.stamp

build/mock_gen.stamp: $(MOCK_INPUTS)
	mockery --config "./.mockery.yaml"
	mkdir -p build
	touch $@

.PHONY: check-gen
check-gen: force-gen gen build/mock_gen.stamp
	# Checking that committed, generated code is up-to-date by ensuring that
	# git reports the files as unchanged after forcibly regenerating the files:
	test -z "$(shell git status --porcelain '**/zgen*' $(STREAM_PYTHON_CLIENT) $(STREAM_TS_CLIENT))" || (git diff; false)

.PHONY: get-deps
get-deps:
	./get-deps.sh

build/stream-gen: cmd/stream-gen/main.go
	go build -o build/stream-gen ./cmd/stream-gen

.PHONY: build
build: export DET_SEGMENT_MASTER_KEY ?=
build: export DET_SEGMENT_WEBUI_KEY ?=
build: export DET_EE_LICENSE_KEY = $(shell cat ../license.txt)
build: export DET_EE_PUBLIC_KEY = $(shell cat ../public.txt)
build: gen
	CGO_ENABLED=0 go build \
		-ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentMasterKey=$(DET_SEGMENT_MASTER_KEY) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentWebUIKey=$(DET_SEGMENT_WEBUI_KEY) \
		          -X github.com/determined-ai/determined/master/internal/license.licenseKey=$(DET_EE_LICENSE_KEY) \
		          -X github.com/determined-ai/determined/master/internal/license.publicKey=$(DET_EE_PUBLIC_KEY)" \
		-o build/determined-master \
		./cmd/determined-master

.PHONY: build-race
build-race: export DET_SEGMENT_MASTER_KEY ?=
build-race: export DET_SEGMENT_WEBUI_KEY ?=
build-race: export DET_EE_LICENSE_KEY = $(shell cat ../license.txt)
build-race: export DET_EE_PUBLIC_KEY = $(shell cat ../public.txt)
build-race: gen
	go build \
		-ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentMasterKey=$(DET_SEGMENT_MASTER_KEY) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentWebUIKey=$(DET_SEGMENT_WEBUI_KEY) \
		          -X github.com/determined-ai/determined/master/internal/license.licenseKey=$(DET_EE_LICENSE_KEY) \
		          -X github.com/determined-ai/determined/master/internal/license.publicKey=$(DET_EE_PUBLIC_KEY)" \
		-o build/determined-master \
		-race \
		./cmd/determined-master

.PHONY: check
check: check-gen
	go mod tidy
	git diff --quiet ../go.mod ../go.sum
	golangci-lint --build-tags integration run --timeout 10m

.PHONY: check-all
check-all: check check-sql

.PHONY: fmt-sql
fmt-sql:
	# there are complaints from this tool that cannot be auto-fixed
	# and we ignore them for now.
	# exclude old migrations before 2023/08/01
	find . -name "*.sql" | grep -Ev 'migrations\/202([0-2].*|30[0-7])' | xargs sqlfluff format -p 0 --dialect postgres > /dev/null || exit 0
	# sqlfluff fix -p 0 --dialect postgres . || exit 0

.PHONY: check-sql
check-sql:
	$(MAKE) fmt-sql
	# Checking that automatically fixable formatting is applied to all sql files
	# by running `make fmt-sql`.
	test -z "$(shell git status --porcelain './static/*')"

.PHONY: fmt
fmt:
	goimports -l -local github.com/determined-ai -w .
	gofumpt -l -w .
	swag fmt

.PHONY: test
test: build/mock_gen.stamp
	gotestsum --junitfile test.junit.xml -- -race -short -coverprofile=coverage.out -covermode atomic -cover ./...

.PHONY: test-intg
test-intg: export DET_INTEGRATION_POSTGRES_URL ?= postgres://postgres:postgres@localhost:5432/determined?sslmode=disable
test-intg: export DET_INTEGRATION_ES_HOST ?= localhost
test-intg: export DET_INTEGRATION_ES_PORT ?= 9200
test-intg: build/mock_gen.stamp
	gotestsum --junitfile test-intg.junit.xml -- -tags=integration -race -coverprofile=coverage.out -covermode atomic -cover ./...

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

.PHONY: pre-package-small
pre-package-small:
	rm -rf build
	mkdir -p build/wheels/
	cp -r ../proto/build/swagger build/swagger
	cp ../harness/dist/*.whl build/wheels/

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

$(LICENSE): $(shell find ../tools/scripts/licenses -type f)

.PHONY: buildx-small
buildx-small:
	docker context rm -f buildx-context-sharedcluster || true
	docker context create buildx-context-sharedcluster
	docker buildx rm -f buildx-build-sharedcluster || true

	docker --context=buildx-context-sharedcluster buildx create --platform "linux/amd64" --bootstrap --use --name buildx-build-sharedcluster;
	docker buildx ls

packaging/LICENSE: $(shell find ../tools/scripts/licenses -type f)
	../tools/scripts/gen-attributions.py master $@

.PHONY: package
package: export DET_SEGMENT_MASTER_KEY ?=
package: export DET_SEGMENT_WEBUI_KEY ?=
package: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
package: gen buildx
	$(GORELEASER) --snapshot --rm-dist

.PHONY: package-dryrun
package-dryrun: export DET_SEGMENT_MASTER_KEY ?=
package-dryrun: export DET_SEGMENT_WEBUI_KEY ?=
package-dryrun: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
package-dryrun: gen buildx
	$(GORELEASER) --snapshot --rm-dist ./.goreleaser_dryrun.yml

.PHONY: package-ee
package-ee: export DET_SEGMENT_MASTER_KEY ?=
package-ee: export DET_SEGMENT_WEBUI_KEY ?=
package-ee: export DET_EE_LICENSE_KEY = $(shell cat ../license.txt)
package-ee: export DET_EE_PUBLIC_KEY = $(shell cat ../public.txt)
package-ee: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)-ee
package-ee: gen buildx
	$(GORELEASER) --snapshot --rm-dist -f ./.goreleaser_ee.yml

.PHONY: package-small
package-small: export DET_SEGMENT_MASTER_KEY ?=
package-small: export DET_SEGMENT_WEBUI_KEY ?=
package-small: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
package-small: gen buildx-small
	$(GORELEASER) --snapshot --rm-dist -f ./.goreleaser_sharedcluster.yml

.PHONY: release
release: export DET_SEGMENT_MASTER_KEY ?=
release: export DET_SEGMENT_WEBUI_KEY ?=
release: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
release: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^v?[0-9.]+$$' | grep "$(VERSION_TAG)" -A1 | sed -n '2 p')
release: gen buildx
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	$(GORELEASER) --rm-dist
	make publish-nvcr

.PHONY: release-dryrun
release-dryrun: export DET_SEGMENT_MASTER_KEY ?=
release-dryrun: export DET_SEGMENT_WEBUI_KEY ?=
release-dryrun: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
release-dryrun: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^v?[0-9.]+$$' | grep "$(VERSION_TAG)" -A1 | sed -n '2 p')
# We intentionally do not invoke `make publish-nvcr(-dryrun)` here.
release-dryrun: gen buildx
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	$(GORELEASER) --rm-dist -f ./.goreleaser_dryrun.yml

.PHONY: release-ee
release-ee: export DET_SEGMENT_MASTER_KEY ?=
release-ee: export DET_SEGMENT_WEBUI_KEY ?=
release-ee: export DET_EE_LICENSE_KEY = $(shell cat ../license.txt)
release-ee: export DET_EE_PUBLIC_KEY = $(shell cat ../public.txt)
release-ee: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)-ee
release-ee: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^v?[0-9.]+-ee$$' | grep "$(GORELEASER_CURRENT_TAG)" -A1 | sed -n '2 p')
release-ee: gen buildx
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	$(GORELEASER) --rm-dist -f ./.goreleaser_ee.yml

.PHONY: release-ee-dryrun
release-ee-dryrun: export DET_SEGMENT_MASTER_KEY ?=
release-ee-dryrun: export DET_SEGMENT_WEBUI_KEY ?=
release-ee-dryrun: export DET_EE_LICENSE_KEY = $(shell cat ../license.txt)
release-ee-dryrun: export DET_EE_PUBLIC_KEY = $(shell cat ../public.txt)
# VERSION_TAG v0.38.0+dryrun becomes v0.38.0-ee+dryrun
release-ee-dryrun: export GORELEASER_CURRENT_TAG := $(shell echo $(VERSION_TAG) | sed "s/+dryrun/-ee&/g")
release-ee-dryrun: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^v?[0-9.]+-ee$$' | grep "$(GORELEASER_CURRENT_TAG)" -A1 | sed -n '2 p')
release-ee-dryrun: gen buildx
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	$(GORELEASER) --rm-dist -f ./.goreleaser_ee_dryrun.yml

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

.PHONY: publish-dev-small
publish-dev-small:
	@$(call manifest_publish, $(DEV_MANIFESTS_SMALL), $(SHARED_CLUSTER_IMAGE))

.PHONY: publish
publish:
	@$(call manifest_publish, $(PUB_MANIFESTS), $(MULTI_ARCH_IMAGES))

.PHONY: publish-dryrun
# Build and upload production images to the determinedai/determined-dryrun
# repository.
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
