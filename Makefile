# THOUGHT: assume this makefile is for local builds and ci would go directly into modules if we want to separate image dependecies
# eg has Java but no Go? I think this + avoiding unnecessary rebuilds at module levels could allow us to simplify our make structure
# and speed up builds too

.PHONY: all
all:
	$(MAKE) get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: get-deps-pip get-deps-master get-deps-bindings get-deps-webui
	$(MAKE) -C agent $@
	$(MAKE) -C proto $@
.PHONY: get-deps-%
get-deps-%:
	$(MAKE) -C $(subst -,/,$*) get-deps
.PHONY: get-deps-pip
get-deps-pip:
	pip install -r requirements.txt

.PHONY: package
package:
	$(MAKE) -C agent $@
	$(MAKE) -C master $@

.PHONY: build-%
build-%:
	$(MAKE) -C $(subst -,/,$*) build

.PHONY: build-docs
build-docs: build-common build-harness build-cli build-deploy build-examples build-helm build-proto
	$(MAKE) -C docs build

.PHONY: build-bindings
build-bindings: build-proto
	$(MAKE) -C bindings build

.PHONY: build-webui
build-webui: build-bindings
	$(MAKE) -C webui build

.PHONY: build-agent
build-agent: build-proto
	$(MAKE) -C agent build

.PHONY: build-master
build-master: build-proto build-webui build-docs
	$(MAKE) -C master build

.PHONY: build
build: build-master build-agent

.PHONY: clean-%
clean-%:
	$(MAKE) -C $(subst -,/,$*) clean
.PHONY: clean
clean: clean-tools clean-proto clean-common clean-harness clean-cli clean-deploy clean-examples clean-docs clean-webui clean-master clean-agent clean-bindings

.PHONY: check-%
check-%:
	$(MAKE) -C $(subst -,/,$*) check
.PHONY: check
check: check-common check-proto check-harness check-cli check-deploy check-e2e_tests check-tools check-master check-webui check-examples check-docs check-schemas
	$(MAKE) check-agent

.PHONY: fmt-%
fmt-%:
	$(MAKE) -C $(subst -,/,$*) fmt
.PHONY: fmt
fmt: fmt-common fmt-harness fmt-cli fmt-deploy fmt-e2e_tests fmt-tools fmt-master fmt-agent fmt-webui fmt-examples fmt-docs fmt-schemas fmt-proto

.PHONY: test-%
test-%:
	$(MAKE) -C $(subst -,/,$*) test
.PHONY: test
test: test-harness test-cli test-common test-master test-agent test-webui
