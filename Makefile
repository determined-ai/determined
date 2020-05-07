export VERSION := $(shell cat VERSION)

export GO111MODULE := on
GOBIN ?= $(shell go env GOPATH)/bin
GORELEASER_VERSION := v0.128.0

BUILDDIR ?= build

# These variables are picked up by GoReleaser for the master build; we default to including no keys.
export DET_SEGMENT_MASTER_KEY ?=
export DET_SEGMENT_WEBUI_KEY ?=

.PHONY: all
all: get-deps build-docker

.PHONY: get-deps
get-deps: python-get-deps go-get-deps
	go get github.com/talos-systems/conform@fa7df19996ece307285da44c73f210c6cbec9207
	$(MAKE) -C webui $@

.PHONY: go-get-deps
go-get-deps:
	$(MAKE) -C master get-deps
	$(MAKE) -C agent get-deps
	curl -fsSL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh -s -- -b $(GOBIN) $(GORELEASER_VERSION)

.PHONY: python-get-deps
python-get-deps:
	pip install -r requirements.txt

.PHONY: package
package:
	$(MAKE) -C agent $@

.PHONY: debs
debs:
	cp -r packaging "$(BUILDDIR)"
	cd "$(BUILDDIR)" && GORELEASER_CURRENT_TAG=$(VERSION) $(GOBIN)/goreleaser -f $(CURDIR)/.goreleaser.yml --snapshot --rm-dist

.PHONY: build
build:
	$(MAKE) -C master $@

.PHONY: build-docker
build-docker: package debs
	$(MAKE) -C master build-docker

.PHONY: clean
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

.PHONY: check
check:
	$(GOBIN)/conform enforce
	$(MAKE) -C cli $@
	$(MAKE) -C common $@
	$(MAKE) -C harness $@
	$(MAKE) -C deploy $@
	$(MAKE) -C e2e_tests $@
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C webui $@

.PHONY: fmt
fmt:
	$(MAKE) -C cli $@
	$(MAKE) -C common $@
	$(MAKE) -C harness $@
	$(MAKE) -C deploy $@
	$(MAKE) -C e2e_tests $@
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C webui $@

.PHONY: test
test:
	$(MAKE) -C harness $@
	$(MAKE) -C cli $@
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C webui $@

# This target assumes that a Hasura instance is running and queries it to
# retrieve the current schema files, producing a schema file that the
# `graphql-python` target can then use to generate code
# without having to have a server running.
.PHONY: graphql-schema
graphql-schema:
	scripts/hasura/export-metadata.sh
	python -m sgqlc.introspection \
		-H "X-Hasura-Admin-Secret: $${DET_HASURA_SECRET:-hasura}" \
		-H "X-Hasura-Role: user" \
		http://localhost:8081/v1/graphql \
		master/graphql-schema.json

.PHONY: graphql-python
graphql-python:
	sgqlc-codegen master/graphql-schema.json common/determined_common/api/gql.py
	black common/determined_common/api/gql.py
	isort common/determined_common/api/gql.py

.PHONY: graphql
graphql:
	$(MAKE) graphql-schema
	$(MAKE) graphql-python
