export VERSION:=$(shell ../version.sh)
export VERSION_TAG:=$(shell ../version.sh -t)

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
release-gh: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
release-gh: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^v?[0-9.]+$$' | grep "$(VERSION_TAG)" -A1 | sed -n '2 p')
release-gh:
	go install github.com/goreleaser/goreleaser@v1.14.1
	git clean -df
	goreleaser --rm-dist

.PHONY: release-gh-rc
release-gh-rc: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
# The following line lists all tags by creation date, finds the current tag and
# the next line after, then prints that second line, which should be the most
# recent previous tag. This works if the previous tag is both a minor release,
# or an rc release. Also, this is a separate make target because it makes it
# easier to compute tag diffs for release notes.
release-gh-rc: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep "$(VERSION_TAG)" -A1 | sed -n '2 p')
release-gh-rc:
	go install github.com/goreleaser/goreleaser@v1.14.1
	git clean -df
	goreleaser --rm-dist

.PHONY: release-gh-dryrun
release-gh-dryrun: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)
release-gh-dryrun: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^v?[0-9.]+$$' | grep "$(VERSION_TAG)" -A1 | sed -n '2 p')
release-gh-dryrun:
	go install github.com/goreleaser/goreleaser@v1.14.1
	git clean -df
	goreleaser --rm-dist -f ./.goreleaser_dryrun.yml

.PHONY: release-gh-ee
release-gh-ee: export GORELEASER_CURRENT_TAG := $(VERSION_TAG)-ee
release-gh-ee: export GORELEASER_PREVIOUS_TAG := $(shell git tag --sort=-creatordate | grep -E '^v?[0-9.]+-ee$$' | grep "$(VERSION_TAG)-ee" -A1 | sed -n '2 p')
release-gh-ee:
	go install github.com/goreleaser/goreleaser@v1.14.1
	git clean -df
	goreleaser --rm-dist

release-gh-ee-dryrun: ;
