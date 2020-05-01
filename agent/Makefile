.PHONY: all \
	clean get-deps \
	build build-docker build-files \
	install-native \
	test \
	check fmt \
	publish publish-dev \

export VERSION:=$(shell cat ../VERSION)

ifeq ($(origin DET_GIT_COMMIT), undefined)
GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_DIRTY := $(if $(shell git status --porcelain),-dirty,)
export DET_GIT_COMMIT := $(GIT_COMMIT)$(GIT_DIRTY)
endif

BUILDDIR ?= ../build

COMPONENT := agent
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

clean:
	rm -rf coverage.out "$(BUILDDIR)"/bin/determined-agent "$(BUILDDIR)"/share/determined/agent/

get-deps:
	cd buildtools && go install golang.org/x/tools/cmd/goimports github.com/rakyll/gotest
	curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/$(GOCI_LINT_VERSION)/install.sh | sh -s -- -b $(GOBIN) $(GOCI_LINT_VERSION)

###################
### Build Phase ###
###################

# Have this here to keep the master and agent symmetric, but nothing actually needs to happen.
build: ;

build-files:
	mkdir -p "$(BUILDDIR)"/share/determined/agent
	rm -rf "$(BUILDDIR)"/share/determined/agent/scripts
	cp -r scripts "$(BUILDDIR)"/share/determined/agent

build-docker: build-files
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t determinedai/determined-$(COMPONENT):$(VERSION) \
		-t $(DOCKER_REGISTRY)determinedai/determined-$(COMPONENT):$(DET_GIT_COMMIT) \
		-t determinedai/determined-dev:determined-$(COMPONENT)-$(DET_GIT_COMMIT) \
		$(EXTRA_TAG) "$(BUILDDIR)" -f Dockerfile

install-native:
	go install -ldflags $(LDFLAGS) ./cmd/determined-agent

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

#####################
### Publish Phase ###
#####################

publish:
	docker push determinedai/determined-$(COMPONENT):$(VERSION)

publish-dev:
	docker push $(DOCKER_REGISTRY)determinedai/determined-$(COMPONENT):$(DET_GIT_COMMIT)
	docker push $(DOCKER_REGISTRY)determinedai/determined-dev:determined-$(COMPONENT)-$(DET_GIT_COMMIT)
