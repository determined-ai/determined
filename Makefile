export VERSION := $(shell ./version.sh)

.PHONY: all
all:
	$(MAKE) get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: get-deps-pip get-deps-go get-deps-webui

.PHONY: get-deps-%
get-deps-%:
	$(MAKE) -C $(subst -,/,$*) get-deps

.PHONY: get-deps-pip
get-deps-pip:
	pip install -r requirements.txt

.PHONY: get-deps-go
get-deps-go:
	$(MAKE) go-version-check
	$(MAKE) get-deps-master
	$(MAKE) get-deps-agent
	$(MAKE) get-deps-proto

# Go versions may look like goM, goM.N, or goM.N.P. Only 1.22.* is supported.
supported_go_minor_version = go1.22
system_go_version := $(shell go version | sed 's/.*\(go[[:digit:]][[:digit:].]*\).*/\1/')
.PHONY: go-version-check
go-version-check:
	@: $(if $(findstring $(supported_go_minor_version), $(system_go_version)), \
				, \
				$(error go version $(system_go_version) not supported. Must use $(supported_go_minor_version).x))

.PHONY: package
package:
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	$(MAKE) -C agent $@
	$(MAKE) -C master $@

.PHONY: package-dryrun
package-dryrun:
	$(MAKE) -C agent $@
	$(MAKE) -C master $@

.PHONY: package-ee
package-ee:
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	$(MAKE) -C agent $@
	$(MAKE) -C master $@

set-config-dev: .git/hooks/pre-commit

.git/hooks/pre-commit:
	pre-commit install

.PHONY: build-%
build-%:
	$(MAKE) -C $(subst -,/,$*) build

.PHONY: build-docs
build-docs: build-harness build-examples build-helm build-proto build-bindings
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
build-master: build-proto
	$(MAKE) -C master build

.PHONY: build
build: build-master build-agent build-webui build-docs

.PHONY: clean-%
clean-%:
	$(MAKE) -C $(subst -,/,$*) clean
.PHONY: clean
clean: clean-tools clean-proto clean-harness clean-examples clean-docs clean-webui clean-master clean-agent clean-bindings

.PHONY: check-%
check-%:
	$(MAKE) -C $(subst -,/,$*) check
.PHONY: check
check: check-proto check-harness check-e2e_tests check-tools check-master check-webui check-examples check-docs check-schemas
	$(MAKE) check-agent

.PHONY: fmt-%
fmt-%:
	$(MAKE) -C $(subst -,/,$*) fmt
.PHONY: fmt
fmt: fmt-harness fmt-e2e_tests fmt-tools fmt-master fmt-agent fmt-webui fmt-examples fmt-docs fmt-schemas fmt-proto

.PHONY: test-%
test-%:
	$(MAKE) -C $(subst -,/,$*) test
.PHONY: test
test: test-harness test-master test-agent test-webui

# local frontend dev server against current DET_MASTER
.PHONY: localfrontend
local: build-bindings get-deps-webui
	HOST="localhost" DET_WEBPACK_PROXY_URL=${DET_MASTER} $(MAKE) -C webui live

.PHONY: devcluster
devcluster:
	devcluster -c tools/devcluster.yaml

TF_LOCK ?= true

.PHONY: slurmcluster
slurmcluster:
	$(MAKE) -C tools/slurm slurmcluster FLAGS="$(FLAGS)" TF_LOCK=$(TF_LOCK)

slurmcluster/usage:
	$(MAKE) -C tools/slurm usage

.PHONY: unslurmcluster
unslurmcluster:
	$(MAKE) -C tools/slurm unslurmcluster TF_LOCK=$(TF_LOCK)
