# Set the path React is going to be served from eg ./ for serving from root.
export PUBLIC_URL ?= /det
export ESLINT_NO_DEV_ERRORS ?= true
source_files := $(shell find src public -type f)

.PHONY: default
default: get-deps
	$(MAKE) build

.PHONY: get-deps-npm
get-deps-npm:
	npm ci --no-audit

node_modules/done.stamp: package-lock.json
	$(MAKE) get-deps-npm
	touch $@

.PHONY: get-deps
get-deps: check-requirements node_modules/done.stamp
	git submodule update --init

.PHONY: build
build: check-requirements build/done.stamp

build/done.stamp: $(source_files) node_modules/done.stamp tsconfig.json craco.config.js jest.config.js
	npm run build
	touch $@

.PHONY: build-storybook
build-storybook:
	npm run storybook:build

.PHONY: clean
clean:
	rm -rf build build-storybook node_modules/

.PHONY: live
live: check-requirements start
.PHONY: start
start:
	npm start

.PHONY: check-js
check-js:
	npm run lint:js
.PHONY: check-css
check-css:
	npm run lint:css
.PHONY: check-package-lock
check-package-lock:
	if grep 'ssh://' package-lock.json ; then echo "ssh url in package-lock.json, please convert to https url" ; false ; fi
.PHONY: check-requirements
check-requirements:
	node --version | grep -E 'v16\.(1[3-9]|2[0-9])' || (echo "node version >=16.13 <17 is required" ; false)
.PHONY: check
check: check-js check-css check-package-lock

.PHONY: fmt-js
fmt-js:
	npm run lint:js -- --fix
.PHONY: fmt-css
fmt-css:
	npm run lint:css -- --fix
.PHONY: fmt-package-lock
fmt-package-lock:
	sed -i '' -e 's|git+ssh://git@github.com|https://github.com|' package-lock.json
.PHONY: fmt
fmt: fmt-js fmt-css fmt-package-lock

.PHONY: test
test:
	npm run test -- --watchAll=false

.PHONY: test-ci
test-ci:
	JEST_JUNIT_OUTPUT_DIR=coverage npm run test:coverage -- --reporters=jest-junit

# shared-web helpers

.PHONY: test-shared
test-shared:
	cd src/shared; ./bin/test.py --repos saas --test-local

.PHONY: push
push-shared: test-shared
	# fail if the branch name is not master
	git rev-parse --abbrev-ref HEAD | grep -qE "^master$\" || (echo "must be on master branch" ; false)

	# fail if there are uncommitted changes.
	git diff --exit-code || (echo "uncommitted changes, please commit or stash" ; false)
	cd $(shell git rev-parse --show-toplevel) && git subtree push --prefix=webui/react/src/shared https://github.com/determined-ai/shared-web main

test-dir-change:
	cd $(shell git rev-parse --show-toplevel) && pwd && ls
	pwd
