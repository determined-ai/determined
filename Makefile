.PHONY: \
  all \
  build \
  build-agent-docker \
  build-docker \
  build-master \
  build-master-docker \
  check \
  clean \
  fmt \
  get-deps \
  graphql \
  graphql-python \
  graphql-schema \
  pin-deps \
  upgrade-deps \
  test

export VERSION := $(shell cat VERSION)
export INTEGRATIONS_HOST_PORT ?= 8080
export DB_HOST_PORT ?= 5433
export INTEGRATIONS_LOG_OPTS := --no-color

GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_DIRTY := $(if $(shell git status --porcelain),-dirty,)
export DET_GIT_COMMIT := $(GIT_COMMIT)$(GIT_DIRTY)

export GO111MODULE := on
GOBIN ?= $(shell go env GOPATH)/bin
GORELEASER_VERSION := v0.128.0

BUILDDIR ?= build

DET_DEV_AGENT_IMAGE := determinedai/determined-dev:determined-agent-$(DET_GIT_COMMIT)
DET_DEV_MASTER_IMAGE := determinedai/determined-dev:determined-master-$(DET_GIT_COMMIT)
export DET_IMAGES := $(DET_DEV_AGENT_IMAGE),$(DET_DEV_MASTER_IMAGE)

# These variables are picked up by GoReleaser for the master build; we default to including no keys.
export DET_SEGMENT_MASTER_KEY ?=
export DET_SEGMENT_WEBUI_KEY ?=

all: get-deps build-docker

get-deps: python-get-deps go-get-deps
	go get github.com/talos-systems/conform@fa7df19996ece307285da44c73f210c6cbec9207
	$(MAKE) -C webui $@

go-get-deps:
	$(MAKE) -C master get-deps
	$(MAKE) -C agent get-deps
	curl -fsSL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh -s -- -b $(GOBIN) $(GORELEASER_VERSION)

python-get-deps:
	pip install -r combined-reqs.txt

build: build-master

build-master:
	$(MAKE) -C master build

debs:
	cp -r packaging "$(BUILDDIR)"
	cd "$(BUILDDIR)" && GORELEASER_CURRENT_TAG=$(VERSION) $(GOBIN)/goreleaser -f $(CURDIR)/.goreleaser.yml --snapshot --rm-dist

build-docker: build debs
	$(MAKE) build-master-docker build-agent-docker

build-agent-docker:
	$(MAKE) -C agent build-docker

build-master-docker:
	$(MAKE) -C master build-docker

clean:
	rm -rf build
	find . \( -name __pycache__ -o -name \*.pyc -o -name .mypy_cache \) -print0 | xargs -0 rm -rf
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C docs $@
	$(MAKE) -C common $@
	$(MAKE) -C harness $@
	$(MAKE) -C cli $@
	$(MAKE) -C deploy $@
	$(MAKE) -C webui $@

# This target assumes that a Hasura instance is running and queries it to
# retrieve the current schema files, producing a schema file that the
# `graphql-python` target can then use to generate code
# without having to have a server running.
graphql-schema:
	scripts/hasura/export-metadata.sh
	python -m sgqlc.introspection \
		-H "X-Hasura-Admin-Secret: $${DET_HASURA_SECRET:-hasura}" \
		-H "X-Hasura-Role: user" \
		http://localhost:8081/v1/graphql \
		master/graphql-schema.json

graphql-python:
	sgqlc-codegen master/graphql-schema.json common/determined_common/api/gql.py
	black common/determined_common/api/gql.py
	isort common/determined_common/api/gql.py

graphql:
	$(MAKE) graphql-schema
	$(MAKE) graphql-python

check: check-commit-messages
	$(MAKE) -C cli $@
	$(MAKE) -C common $@
	$(MAKE) -C harness $@
	$(MAKE) -C deploy $@
	$(MAKE) -C e2e_tests $@
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C webui $@

check-commit-messages:
	$(GOBIN)/conform enforce

fmt:
	$(MAKE) -C cli $@
	$(MAKE) -C common $@
	$(MAKE) -C harness $@
	$(MAKE) -C deploy $@
	$(MAKE) -C e2e_tests $@
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C webui $@

test:
	$(MAKE) -C harness $@
	$(MAKE) -C cli $@
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C webui $@

