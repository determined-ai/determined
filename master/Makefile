.DEFAULT_GOAL := build
SHELL := bash
LICENSE := packaging/LICENSE
SCHEMA_INPUTS = ../schemas/gen.py $(shell find ./pkg/schemas/ -name 'zgen_*.go' -prune -o -name '*.go' -print) $(shell find ../schemas/expconf -name '*.json')
STREAM_INPUTS = $(shell find ./internal/stream/ -name '*_test.go' -prune -o -name '*.go' -print)
STREAM_PYTHON_CLIENT = ../harness/determined/common/streams/wire.py
STREAM_TS_CLIENT = ../webui/react/src/services/stream/wire.ts
MOCK_INPUTS = ./internal/sproto/task.go ./internal/db/database.go ./internal/command/authz_iface.go ../go.mod ../go.sum ./internal/rm/resource_manager_iface.go ./internal/task/allocation_service_iface.go
GORELEASER = goreleaser

export VERSION := $(shell cat ../VERSION)
export GO111MODULE := on
export DOCKER_REPO ?= determinedai

FULL_COMMIT = $(shell git rev-parse HEAD)
SHORT_COMMIT = $(shell git rev-parse HEAD | head -c9)
PROJECT_NAME = determined-master-release-testing
EE_PROJECT_NAME = hpe-mlde-master-release-testing
ARCHS = amd64 arm64 ppc64
ARCH_SMALL = amd64-shared-cluster
MULTI_ARCH_IMAGES = $(shell for arch in $(ARCHS); do echo $(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT)-$$arch; done)
EE_MULTI_ARCH_IMAGES = $(shell for arch in $(ARCHS); do echo $(DOCKER_REPO)/$(EE_PROJECT_NAME):$(FULL_COMMIT)-$$arch; done)
SHARED_CLUSTER_IMAGE=$(shell echo $(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT)-$(ARCH_SMALL))

NVCR_REPO ?= nvcr.io/isv-ngc-partner/determined

PUB_MANIFESTS = \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT) \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(SHORT_COMMIT) \
	$(DOCKER_REPO)/$(PROJECT_NAME):$(VERSION)

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
	mockery --quiet --name=DB --dir=internal/db --output internal/mocks --filename db.go
	mockery --quiet --name=Resources --dir=internal/sproto --output internal/mocks --filename resources.go
	mockery --quiet --name=UserAuthZ --dir=internal/user --output internal/mocks --filename user_authz_iface.go
	mockery --quiet --name=WorkspaceAuthZ --dir=internal/workspace --output internal/mocks --filename workspace_authz_iface.go
	mockery --quiet --name=ProjectAuthZ --dir=internal/project --output internal/mocks --filename project_authz_iface.go
	mockery --quiet --name=ExperimentAuthZ --dir=internal/experiment --output internal/mocks --filename authz_experiment_iface.go
	mockery --quiet --name=NSCAuthZ --dir=internal/command --output internal/mocks --filename nsc_authz_iface.go
	mockery --quiet --name=ModelAuthZ --dir=internal/model --output internal/mocks --filename authz_model_iface.go
	mockery --quiet --name=PodInterface --srcpkg=k8s.io/client-go/kubernetes/typed/core/v1 --output internal/mocks --filename pod_iface.go
	mockery --quiet --name=EventInterface --srcpkg=k8s.io/client-go/kubernetes/typed/core/v1 --output internal/mocks --filename event_iface.go
	mockery --quiet --name=NodeInterface --srcpkg=k8s.io/client-go/kubernetes/typed/core/v1 --output internal/mocks --filename node_iface.go
	mockery --quiet --name=ResourceManager --dir=internal/rm --output internal/mocks --filename rm.go
	mockery --quiet --name=AllocationService --dir=internal/task --output internal/mocks/allocationmocks --filename allocation_service.go --outpkg allocationmocks
	mockery --quiet --name=ResourceManagerAuthZ --dir=internal/rm --output internal/mocks --filename rm_authz_iface.go
	mockery --quiet --name=Interface --output internal/mocks --srcpkg "k8s.io/client-go/kubernetes" --filename k8s_clientset.go --structname K8sClientsetInterface
	mockery --quiet --name=CoreV1Interface --output internal/mocks --srcpkg "k8s.io/client-go/kubernetes/typed/core/v1" --filename k8s_corev1_iface.go --structname K8sCoreV1Interface

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
			[ $$arch == "ppc64" ] && arch="ppc64le"; \
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
package: export GORELEASER_CURRENT_TAG := $(VERSION)
package: gen buildx
	$(GORELEASER) --snapshot --rm-dist 

.PHONY: package-ee
package-ee: export DET_SEGMENT_MASTER_KEY ?=
package-ee: export DET_SEGMENT_WEBUI_KEY ?=
package-ee: export DET_EE_LICENSE_KEY = $(shell cat ../license.txt)
package-ee: export DET_EE_PUBLIC_KEY = $(shell cat ../public.txt)
package-ee: export GORELEASER_CURRENT_TAG := $(VERSION)-ee
package-ee: gen buildx
	$(GORELEASER) --snapshot --rm-dist -f ./.goreleaser_ee.yml

.PHONY: package-small
package-small: export DET_SEGMENT_MASTER_KEY ?=
package-small: export DET_SEGMENT_WEBUI_KEY ?=
package-small: export GORELEASER_CURRENT_TAG := $(VERSION)
package-small: gen buildx-small
	$(GORELEASER) --snapshot --rm-dist -f ./.goreleaser_sharedcluster.yml

.PHONY: release
release: export DET_SEGMENT_MASTER_KEY ?=
release: export DET_SEGMENT_WEBUI_KEY ?=
release: export GORELEASER_CURRENT_TAG := $(VERSION)
release: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^[0-9.]+$$' | grep "$(VERSION)" -A1 | sed -n '2 p')
release: gen buildx
	$(GORELEASER) --rm-dist
	make publish-nvcr

.PHONY: release-ee
release-ee: export DET_SEGMENT_MASTER_KEY ?=
release-ee: export DET_SEGMENT_WEBUI_KEY ?=
release-ee: export DET_EE_LICENSE_KEY = $(shell cat ../license.txt)
release-ee: export DET_EE_PUBLIC_KEY = $(shell cat ../public.txt)
release-ee: export GORELEASER_CURRENT_TAG := $(VERSION)-ee
release-ee: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^[0-9.]+-ee$$' | grep "$(VERSION)-ee" -A1 | sed -n '2 p')
release-ee: gen buildx
	$(GORELEASER) --rm-dist -f ./.goreleaser_ee.yml

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

.PHONY: publish-ee
publish-ee:
	@$(call manifest_publish, $(EE_PUB_MANIFESTS), $(EE_MULTI_ARCH_IMAGES))

.PHONY: publish-nvcr
publish-nvcr:
	for image in $(NVCR_TAGS); do \
		docker tag $(DOCKER_REPO)/$(PROJECT_NAME):$(FULL_COMMIT)-amd64 $$image; \
		docker push $$image; \
	done
