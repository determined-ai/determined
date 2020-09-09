.PHONY: all
all:
	$(MAKE) get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps:
	pip install -r requirements.txt
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C proto $@
	$(MAKE) -C webui $@

.PHONY: package
package:
	$(MAKE) -C agent $@
	$(MAKE) -C master $@

.PHONY: build-%
build-%:
	$(MAKE) -C $(subst -,/,$*) build
.PHONY: build-docs
build-docs: build-common build-harness build-cli build-deploy build-examples build-helm
	$(MAKE) -C docs build
.PHONY: build-master
build-master: build-webui build-docs
	$(MAKE) -C master build
.PHONY: build-webui
build-webui: build-proto
	$(MAKE) -C webui build
.PHONY: build
build: build-master build-agent

.PHONY: clean-%
clean-%:
	$(MAKE) -C $(subst -,/,$*) clean
.PHONY: clean
clean: clean-tools clean-proto clean-common clean-harness clean-cli clean-deploy clean-examples clean-docs clean-webui clean-master clean-agent

.PHONY: check-%
check-%:
	$(MAKE) -C $(subst -,/,$*) check
.PHONY: check
check: check-common check-proto check-harness check-cli check-deploy check-e2e_tests check-tools check-master check-agent check-webui check-examples

.PHONY: fmt-%
fmt-%:
	$(MAKE) -C $(subst -,/,$*) fmt
.PHONY: fmt
fmt: fmt-common fmt-harness fmt-cli fmt-deploy fmt-e2e_tests fmt-tools fmt-master fmt-agent fmt-webui fmt-examples fmt-docs

.PHONY: test-%
test-%:
	$(MAKE) -C $(subst -,/,$*) test
.PHONY: test
test: test-harness test-cli test-master test-agent test-webui
