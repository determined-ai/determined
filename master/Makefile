.PHONY: all \
	clean get-deps \
	build build-docs build-docker build-elm build-files build-python-packages build-react build-static build-webui \
	install-native \
	test \
	check fmt \
	publish publish-dev

export VERSION := $(shell cat ../VERSION)

ifeq ($(origin DET_GIT_COMMIT), undefined)
GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_DIRTY := $(if $(shell git status --porcelain),-dirty,)
export DET_GIT_COMMIT := $(GIT_COMMIT)$(GIT_DIRTY)
endif

BUILDDIR ?= ../build

export GO111MODULE := on

all: clean get-deps build

######################
### Validate Phase ###
######################

clean: clean-python-packages
	rm -rf coverage.out "$(BUILDDIR)"/bin/determined-master "$(BUILDDIR)"/share/determined/master/

clean-python-packages:
	$(MAKE) -C ../common clean
	$(MAKE) -C ../cli clean
	$(MAKE) -C ../harness clean

get-deps:
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go install golang.org/x/tools/cmd/goimports
	go install github.com/goreleaser/goreleaser

###################
### Build Phase ###
###################

build: build-docs build-elm build-files build-python-packages build-react

build-files:
	mkdir -p "$(BUILDDIR)"/share/determined/master
	rm -rf "$(BUILDDIR)"/share/determined/master/static
	cp -r static "$(BUILDDIR)"/share/determined/master

# This depends on `build-python-packages` for the sake of parallel builds: the
# `build-python-packages` target depends on the `clean-python-packages` target, which removes
# .egg-info/ directories that Sphinx will need to import Determined things. Running both at the same time
# can cause Sphinx to fail.
build-docs: build-python-packages build-examples
	mkdir -p $(BUILDDIR)/share/determined/master/webui/docs
	$(MAKE) -C ../docs build
	cp -r ../docs/site/html/* $(BUILDDIR)/share/determined/master/webui/docs

build-examples:
	$(MAKE) -C ../examples build

build-elm:
	$(MAKE) -C ../webui/elm build

build-python-packages: clean-python-packages
	mkdir -p $(BUILDDIR)/share/determined/master/wheels/
	$(MAKE) -C ../common build
	cp ../common/dist/*.whl $(BUILDDIR)/share/determined/master/wheels/
	$(MAKE) -C ../cli build
	cp ../cli/dist/*.whl $(BUILDDIR)/share/determined/master/wheels/
	$(MAKE) -C ../harness build
	cp ../harness/dist/*.whl $(BUILDDIR)/share/determined/master/wheels/

build-react:
	mkdir -p $(BUILDDIR)/share/determined/master/webui/react
	$(MAKE) -C ../webui/react build
	cp -r ../webui/react/build/* $(BUILDDIR)/share/determined/master/webui/react

build-docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t determinedai/determined-master:$(VERSION) \
		-t determinedai/determined-master:$(DET_GIT_COMMIT) \
		-t determinedai/determined-dev:determined-master-$(DET_GIT_COMMIT) \
		$(EXTRA_TAG) "$(BUILDDIR)" -f Dockerfile

install-native:
	go install -ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION)" ./cmd/determined-master

####################
### Verify Phase ###
####################

check:
	golangci-lint run

fmt:
	goimports -l -local github.com/determined-ai -w .

##################
### Test Phase ###
##################

test:
	go test -v -short -coverprofile=coverage.out -covermode count -cover ./...

#####################
### Publish Phase ###
#####################

publish:
	docker push determinedai/determined-master:$(VERSION)

publish-dev:
	docker push determinedai/determined-master:$(DET_GIT_COMMIT)
	docker push determinedai/determined-dev:determined-master-$(DET_GIT_COMMIT)
