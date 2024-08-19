export VERSION:=$(shell ../version.sh)

build/stamp: $(shell find charts -type f)
	mkdir -p build
	rm -rf build/*.tgz
	helm package --destination build charts/determined --version $(VERSION) --app-version $(VERSION)
	cd build/ && ln -s determined-*.tgz determined-latest.tgz && cd ..
	touch $@

.PHONY: build
build: build/stamp

fmt:
	helm lint charts/determined/

.PHONY: clean
clean:
	rm -rf build/

.PHONY: release-gh
release-gh: export GORELEASER_CURRENT_TAG := $(VERSION)
release-gh: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^[0-9.]+$$' | grep "$(VERSION)" -A1 | sed -n '2 p')
release-gh:
	go install github.com/goreleaser/goreleaser@v1.14.1
	git clean -df
	goreleaser --rm-dist

.PHONY: release-gh-dryrun
release-gh-dryrun: export GORELEASER_CURRENT_TAG := $(VERSION)
release-gh-dryrun: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^[0-9.]+$$' | grep "$(VERSION)" -A1 | sed -n '2 p')
release-gh-dryrun:
	go install github.com/goreleaser/goreleaser@v1.14.1
	git clean -df
	goreleaser --rm-dist -f ./.goreleaser_dryrun.yml

.PHONY: release-gh-ee
release-gh-ee: export GORELEASER_CURRENT_TAG := $(VERSION)-ee
release-gh-ee: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^[0-9.]+-ee$$' | grep "$(VERSION)-ee" -A1 | sed -n '2 p')
release-gh-ee:
	go install github.com/goreleaser/goreleaser@v1.14.1
	git clean -df
	goreleaser --rm-dist

release-gh-ee-dryrun: ;
