.PHONY: all \
	clean get-deps \
	build build-docs build-docker build-elm build-files build-python-packages build-react build-static build-webui \
	install-native \
	test test-integrations \
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
LDFLAGS := "-X github.com/determined-ai/determined/$(COMPONENT)/version.Version=$(VERSION)"
GOTEST := $(if $(shell which gotest 2>/dev/null),gotest,go test)
export GO111MODULE := on

GOBIN ?= $(shell go env GOPATH)/bin
GOCI_LINT_VERSION := v1.20.0

DOCKER_REGISTRY ?=
ifneq ($(DOCKER_REGISTRY),)
	EXTRA_TAG := -t $(DOCKER_REGISTRY)determinedai/determined-dev:determined-$(COMPONENT)-$(DET_GIT_COMMIT)
endif

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
	  && go install golang.org/x/tools/cmd/goimports github.com/rakyll/gotest
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
	$(MAKE) -C ../common build
	$(MAKE) -C ../cli build
	$(MAKE) -C ../harness build

build-react:
	$(MAKE) -C ../webui/react build

build-docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t determinedai/determined-$(COMPONENT):$(VERSION) \
		-t $(DOCKER_REGISTRY)determinedai/determined-$(COMPONENT):$(DET_GIT_COMMIT) \
		-t determinedai/determined-dev:determined-$(COMPONENT)-$(DET_GIT_COMMIT) \
		$(EXTRA_TAG) "$(BUILDDIR)" -f Dockerfile

install-native:
	go install -ldflags $(LDFLAGS) ./cmd/determined-master

####################
### Verify Phase ###
####################

check:
	$(GOBIN)/golangci-lint run
	@# Since our custom goimports is more strict than golangci-lint's, run
	@# the check separately.
	@echo "Running goimports linter"
	@[ -z "$$( $(GOBIN)/goimports -l -local github.com/determined-ai . )" ] || (echo please make fmt files && exit 1)

fmt:
	$(GOBIN)/goimports -l -local github.com/determined-ai -w .

##################
### Test Phase ###
##################

test:
	$(GOTEST) -ldflags $(LDFLAGS) -v -short -coverprofile=coverage.out -covermode count -cover ./...

test-cloud-integrations:
	$(GOTEST) -ldflags $(LDFLAGS) -v -run Cloud -coverprofile=coverage.out -covermode count -cover ./...

test-integrations:
	$(GOTEST) -ldflags $(LDFLAGS) -v -run Integration -coverprofile=coverage.out -covermode count -cover ./...

#####################
### Publish Phase ###
#####################

publish:
	docker push determinedai/determined-$(COMPONENT):$(VERSION)

publish-dev:
	docker push $(DOCKER_REGISTRY)determinedai/determined-$(COMPONENT):$(DET_GIT_COMMIT)
	docker push $(DOCKER_REGISTRY)determinedai/determined-dev:determined-$(COMPONENT)-$(DET_GIT_COMMIT)
