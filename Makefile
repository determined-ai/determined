.PHONY: all
all: get-deps build

.PHONY: get-deps
get-deps:
	GO111MODULE=on go get github.com/talos-systems/conform@fa7df19996ece307285da44c73f210c6cbec9207
	pip install -r requirements.txt
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C webui $@

.PHONY: package
package:
	$(MAKE) -C agent $@
	$(MAKE) -C master $@

.PHONY: build-%
build-%:
	$(MAKE) -C $(subst -,/,$(@:build-%=%)) build
.PHONY: build-docs
build-docs: build-common build-harness build-cli build-deploy build-examples
	$(MAKE) -C docs build
.PHONY: build-master
build-master: build-docs build-webui-elm build-webui-react
	$(MAKE) -C master build
.PHONY: build
build: build-master build-agent

.PHONY: clean-%
clean-%:
	$(MAKE) -C $(subst -,/,$(@:clean-%=%)) clean
.PHONY: clean
clean: clean-tools clean-common clean-harness clean-cli clean-deploy clean-examples clean-docs clean-webui clean-master clean-agent

.PHONY: check-%
check-%:
	$(MAKE) -C $(subst -,/,$(@:check-%=%)) check
.PHONY: check
check: check-common check-harness check-cli check-deploy check-e2e_tests check-master check-agent check-webui
	conform enforce

.PHONY: fmt-%
fmt-%:
	$(MAKE) -C $(subst -,/,$(@:fmt-%=%)) fmt
.PHONY: fmt
fmt: fmt-common fmt-harness fmt-cli fmt-deploy fmt-e2e_tests fmt-master fmt-agent fmt-webui

.PHONY: test-%
test-%:
	$(MAKE) -C $(subst -,/,$(@:test-%=%)) test
.PHONY: test
test: test-harness test-cli test-master test-agent test-webui

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
