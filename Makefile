export VERSION := $(shell cat VERSION)

.PHONY: all
all: get-deps package

.PHONY: get-deps
get-deps:
	go get github.com/talos-systems/conform@fa7df19996ece307285da44c73f210c6cbec9207
	pip install -r requirements.txt
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C webui $@

.PHONY: package
package:
	$(MAKE) -C agent $@
	$(MAKE) -C master $@

.PHONY: build
build:
	$(MAKE) -C common $@
	$(MAKE) -C harness $@
	$(MAKE) -C cli $@
	$(MAKE) -C deploy $@
	$(MAKE) -C examples $@
	$(MAKE) -C docs $@
	$(MAKE) -C webui $@
	$(MAKE) -C master $@
	$(MAKE) -C agent $@

.PHONY: clean
clean:
	$(MAKE) -C examples $@
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
	conform enforce
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
