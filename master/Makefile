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

COMPONENT := master
export GO111MODULE := on

GOBIN ?= $(shell go env GOPATH)/bin
GOCI_LINT_VERSION := v1.20.0

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
	cd buildtools \
	  && go install golang.org/x/tools/cmd/goimports
	curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/$(GOCI_LINT_VERSION)/install.sh | sh -s -- -b $(GOBIN) $(GOCI_LINT_VERSION)

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
build-docs: build-python-packages
	$(MAKE) -C ../docs build

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
	$(MAKE) -C ../webui/react build

build-docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t determinedai/determined-$(COMPONENT):$(VERSION) \
		-t determinedai/determined-$(COMPONENT):$(DET_GIT_COMMIT) \
		-t determinedai/determined-dev:determined-$(COMPONENT)-$(DET_GIT_COMMIT) \
		$(EXTRA_TAG) "$(BUILDDIR)" -f Dockerfile

install-native:
	go install -ldflags "-X github.com/determined-ai/determined/$(COMPONENT)/version.Version=$(VERSION)" ./cmd/determined-master

####################
### Verify Phase ###
####################

check:
	$(GOBIN)/golangci-lint run

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
	docker push determinedai/determined-$(COMPONENT):$(VERSION)

publish-dev:
	docker push determinedai/determined-$(COMPONENT):$(DET_GIT_COMMIT)
	docker push determinedai/determined-dev:determined-$(COMPONENT)-$(DET_GIT_COMMIT)
