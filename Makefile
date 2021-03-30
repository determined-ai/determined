.PHONY: all
all:
	$(MAKE) get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: get-deps-pip get-deps-go get-deps-bindings get-deps-webui

.PHONY: get-deps-%
get-deps-%:
	$(MAKE) -C $(subst -,/,$*) get-deps

.PHONY: get-deps-pip
get-deps-pip:
	pip install -r requirements.txt

.PHONY: get-deps-go
get-deps-go:
	$(MAKE) get-deps-master
	$(MAKE) get-deps-agent
	$(MAKE) get-deps-proto

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
check: check-common check-proto check-harness check-cli check-deploy check-e2e_tests check-tools check-master check-webui check-examples check-docs check-schemas check-examples
	$(MAKE) check-agent

.PHONY: fmt-%
fmt-%:
	$(MAKE) -C $(subst -,/,$*) fmt
.PHONY: fmt
fmt: fmt-common fmt-harness fmt-cli fmt-deploy fmt-e2e_tests fmt-tools fmt-master fmt-agent fmt-webui fmt-examples fmt-docs fmt-schemas fmt-proto fmt-examples

.PHONY: test-%
test-%:
	$(MAKE) -C $(subst -,/,$*) test
.PHONY: test
test: test-harness test-cli test-common test-master test-agent test-webui
