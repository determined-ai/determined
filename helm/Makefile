export VERSION:=$(shell cat ../VERSION)

build/stamp: $(shell find charts -type f)
	mkdir -p build
	rm -rf build/*.tgz
	helm package --destination build charts/determined
	cd build/ && ln -s determined-*.tgz determined-latest.tgz && cd ..
	touch $@

.PHONY: build
build: build/stamp

fmt:
	helm lint charts/determined/

.PHONY: clean
clean:
	rm -rf build/

.PHONY: release
release: export GORELEASER_CURRENT_TAG := $(VERSION)
release: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -v "rc" | grep "v$(VERSION)" -A1 | sed -n '2 p')
release: export NGC_REPO := https://helm.ngc.nvidia.com/isv-ngc-partner/determined
release: export NGC_API_USERNAME ?=
release: export NGC_API_KEY ?=
release:
	goreleaser --rm-dist
	helm repo add determined $$NGC_REPO --username $$NGC_API_USERNAME --password $$NGC_API_KEY
	helm cm-push -f build/determined-latest.tgz determined
