# Set the path React is going to be served from eg ./ for serving from root.
export PUBLIC_URL ?= /det
export ESLINT_NO_DEV_ERRORS ?= true
source_files := $(shell find src public -type f)

.PHONY: default
default: get-deps
	$(MAKE) build


.PHONY: get-deps-npm
get-deps-npm:
	for i in `seq 3`; do \
		npm ci --no-audit && break ; \
	done

node_modules/done.stamp: package-lock.json
	$(MAKE) get-deps-npm
	touch $@

.PHONY: get-deps
get-deps: node_modules/done.stamp

.PHONY: build
build: build/done.stamp

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
live:
	SERVER_ADDRESS=http://localhost:8080 npm start

.PHONY: check-js
check-js:
	npm run lint:js
.PHONY: check-css
check-css:
	npm run lint:css
.PHONY: check
check: check-js check-css

.PHONY: fmt-js
fmt-js:
	npm run lint:js -- --fix
.PHONY: fmt-css
fmt-css:
	npm run lint:css -- --fix
.PHONY: fmt
fmt: fmt-js fmt-css

.PHONY: test
test:
	npm run test -- --watchAll=false

.PHONY: test-ci
test-ci:
	JEST_JUNIT_OUTPUT_DIR=coverage npm run test:coverage -- --watchAll=false --reporters=jest-junit
