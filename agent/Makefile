.PHONY: all \
	clean get-deps \
	build-docker build-files \
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

export GO111MODULE := on

all: clean get-deps

######################
### Validate Phase ###
######################

clean:
	rm -rf coverage.out "$(BUILDDIR)"/bin/determined-agent "$(BUILDDIR)"/share/determined/agent/

get-deps:
	go mod download

###################
### Build Phase ###
###################

build-files:
	mkdir -p "$(BUILDDIR)"/share/determined/agent
	rm -rf "$(BUILDDIR)"/share/determined/agent/scripts
	cp -r scripts "$(BUILDDIR)"/share/determined/agent

build-docker: build-files
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t determinedai/determined-agent:$(VERSION) \
		-t determinedai/determined-agent:$(DET_GIT_COMMIT) \
		-t determinedai/determined-dev:determined-agent-$(DET_GIT_COMMIT) \
		$(EXTRA_TAG) "$(BUILDDIR)" -f Dockerfile

install-native:
	go install -ldflags "-X github.com/determined-ai/determined/agent/version.Version=$(VERSION)" ./cmd/determined-agent

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
	docker push determinedai/determined-agent:$(VERSION)

publish-dev:
	docker push determinedai/determined-agent:$(DET_GIT_COMMIT)
	docker push determinedai/determined-dev:determined-agent-$(DET_GIT_COMMIT)
