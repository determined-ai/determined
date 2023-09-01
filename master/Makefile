.DEFAULT_GOAL := build
SHELL := bash
GENERATED := packaging/LICENSE
GENERATION_INPUTS = ../schemas/gen.py $(shell find ./pkg/schemas/ -name 'zgen_*.go' -prune -o -name '*.go' -print) $(shell find ../schemas/expconf -name '*.json')
MOCK_GENERATION_INPUTS = ./internal/sproto/task.go ./internal/db/database.go ./internal/command/authz_iface.go ./go.mod ./go.sum ./internal/rm/resource_manager_iface.go ./internal/task/allocation_service_iface.go
GORELEASER = goreleaser

export VERSION := $(shell cat ../VERSION)
export GO111MODULE := on
export DOCKER_REPO ?= determinedai

FULL_COMMIT = $(shell git rev-parse HEAD)
SHORT_COMMIT = $(shell git rev-parse HEAD | head -c9)
PROJECT_NAME = determined-master
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
clean: ungen
	rm -rf dist
	rm -rf build

.PHONY: ungen
ungen:
	rm -f $(GENERATED)
	rm -f `find ./pkg/schemas/ -name 'zgen_*.go'` build/schema_gen.stamp
	rm -f `find ./internal/mocks -name '*.go'` build/mock_gen.stamp

.PHONY: gen
gen: $(GENERATED) build/schema_gen.stamp

.PHONY: force-gen
force-gen:
	rm -f build/schema_gen.stamp

build/schema_gen.stamp: $(GENERATION_INPUTS)
	go generate ./pkg/schemas/...
	mkdir -p build
	touch $@

build/mock_gen.stamp: $(MOCK_GENERATION_INPUTS)
	mockery --name=DB --dir=internal/db --output internal/mocks --filename db.go
	mockery --name=Resources --dir=internal/sproto --output internal/mocks --filename resources.go
	mockery --name=UserAuthZ --dir=internal/user --output internal/mocks --filename user_authz_iface.go
	mockery --name=WorkspaceAuthZ --dir=internal/workspace --output internal/mocks --filename workspace_authz_iface.go
	mockery --name=ProjectAuthZ --dir=internal/project --output internal/mocks --filename project_authz_iface.go
	mockery --name=ExperimentAuthZ --dir=internal/experiment --output internal/mocks --filename authz_experiment_iface.go
	mockery --name=NSCAuthZ --dir=internal/command --output internal/mocks --filename nsc_authz_iface.go
	mockery --name=ModelAuthZ --dir=internal/model --output internal/mocks --filename authz_model_iface.go
	mockery --name=PodInterface --srcpkg=k8s.io/client-go/kubernetes/typed/core/v1 --output internal/mocks --filename pod_iface.go     
	mockery --name=EventInterface --srcpkg=k8s.io/client-go/kubernetes/typed/core/v1 --output internal/mocks --filename event_iface.go 
	mockery --name=NodeInterface --srcpkg=k8s.io/client-go/kubernetes/typed/core/v1 --output internal/mocks --filename node_iface.go 
	mockery --name=ResourceManager --dir=internal/rm --output internal/mocks --filename rm.go
	mockery --name=AllocationService --dir=internal/task --output internal/mocks/allocationmocks --filename allocation_service.go --outpkg allocationmocks
	mockery --name=ResourceManagerAuthZ --dir=internal/rm --output internal/mocks --filename rm_authz_iface.go

	# fmt once since the mocks are not generated formatted.
	goimports -l -local github.com/determined-ai -w ./internal/mocks ./internal/mocks/allocationmocks
	mkdir -p build
	touch $@

.PHONY: check-gen
check-gen: force-gen gen build/mock_gen.stamp
	# Checking that committed, generated code is up-to-date by ensuring that
	# git reports the files as unchanged after forcibly regenerating the files:
	test -z "$(shell git status --porcelain '**/zgen*')"
	test -z "$(shell git status --porcelain './internal/mocks/*')"

.PHONY: get-deps
get-deps:
	./get-deps.sh

.PHONY: build
build: export DET_SEGMENT_MASTER_KEY ?=
build: export DET_SEGMENT_WEBUI_KEY ?=
build: gen
	go build \
		-ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentMasterKey=$(DET_SEGMENT_MASTER_KEY) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentWebUIKey=$(DET_SEGMENT_WEBUI_KEY)" \
		-o build/determined-master \
		./cmd/determined-master

.PHONY: build-race
build-race: export DET_SEGMENT_MASTER_KEY ?=
build-race: export DET_SEGMENT_WEBUI_KEY ?=
build-race: gen
	go build \
		-ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentMasterKey=$(DET_SEGMENT_MASTER_KEY) \
		          -X github.com/determined-ai/determined/master/internal/config.DefaultSegmentWebUIKey=$(DET_SEGMENT_WEBUI_KEY)" \
		-o build/determined-master \
		-race \
		./cmd/determined-master

.PHONY: check
check: check-gen
	go mod tidy
	git diff --quiet go.mod go.sum
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
	gotestsum --junitfile test-intg.junit.xml -- -race -tags=integration -coverprofile=coverage.out -covermode atomic -cover ./...

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
	../tools/scripts/gen-attributions.py master $@

.PHONY: package
package: export DET_SEGMENT_MASTER_KEY ?=
package: export DET_SEGMENT_WEBUI_KEY ?=
package: export GORELEASER_CURRENT_TAG := $(VERSION)
package: gen buildx
	$(GORELEASER) --snapshot --rm-dist

.PHONY: release
release: export DET_SEGMENT_MASTER_KEY ?=
release: export DET_SEGMENT_WEBUI_KEY ?=
release: export GORELEASER_CURRENT_TAG := $(VERSION)
release: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^[0-9.]+$$' | grep "$(VERSION)" -A1 | sed -n '2 p')
release: gen buildx
	$(GORELEASER) --rm-dist
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
