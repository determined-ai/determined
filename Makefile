.PHONY: all
all:
	$(MAKE) get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps:
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
