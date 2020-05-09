.PHONY: all \
	clean get-deps \
	build build-docs build-docker build-elm build-files build-python-packages build-react build-static build-webui \
	install \
	test \
	check fmt \
	publish publish-dev

export VERSION := $(shell cat ../VERSION)
export GO111MODULE := on

# These variables are picked up by GoReleaser for the master build; we default to including no keys.
export DET_SEGMENT_MASTER_KEY ?=
export DET_SEGMENT_WEBUI_KEY ?=

all: clean get-deps build

######################
### Validate Phase ###
######################

clean: clean-python-packages
	rm -rf coverage.out build/bin/determined-master build/share/determined/master/

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
	go build \
		-ldflags "-X github.com/determined-ai/determined/master/version.Version=$(VERSION)" \
		-o build/determined-master \
		./cmd/determined-master

build-files:
	mkdir -p build/share/determined/master
	rm -rf build/share/determined/master/static
	cp -r static build/share/determined/master

# This depends on `build-python-packages` for the sake of parallel builds: the
# `build-python-packages` target depends on the `clean-python-packages` target, which removes
# .egg-info/ directories that Sphinx will need to import Determined things. Running both at the same time
# can cause Sphinx to fail.
build-docs: build-python-packages build-examples
	mkdir -p build/share/determined/master/webui/docs
	$(MAKE) -C ../docs build
	cp -r ../docs/site/html/* build/share/determined/master/webui/docs

build-examples:
	$(MAKE) -C ../examples build

build-elm:
	mkdir -p build/share/determined/master/webui/elm
	$(MAKE) -C ../webui/elm build
	cp -r ../webui/elm/public build/share/determined/master/webui/elm

build-python-packages: clean-python-packages
	mkdir -p build/share/determined/master/wheels/
	$(MAKE) -C ../common build
	cp ../common/dist/*.whl build/share/determined/master/wheels/
	$(MAKE) -C ../cli build
	cp ../cli/dist/*.whl build/share/determined/master/wheels/
	$(MAKE) -C ../harness build
	cp ../harness/dist/*.whl build/share/determined/master/wheels/

build-react:
	mkdir -p build/share/determined/master/webui/react
	$(MAKE) -C ../webui/react build
	cp -r ../webui/react/build/* build/share/determined/master/webui/react

install:
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

package:
	GORELEASER_CURRENT_TAG=$(VERSION) goreleaser --snapshot --rm-dist

publish:
	docker push determinedai/determined-master:$(VERSION)

publish-dev:
	docker push determinedai/determined-master:$(shell git rev-parse HEAD)
	docker push determinedai/determined-dev:determined-master-$(shell git rev-parse HEAD)
