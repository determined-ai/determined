# Set the path React is going to be served from eg ./ for serving from root.
export PUBLIC_URL ?= /det
export ESLINT_NO_DEV_ERRORS ?= true
swagger_client := src/services/swagger
source_files := $(shell find src public -type f)
ts_bindings_src := ../../bindings/build/typescript-fetch
ts_bindings_dest := src/services/api-ts-sdk
ts_bindings := $(shell find $(ts_bindings_src) -type f)

.PHONY: default
default: get-deps
	$(MAKE) build


.PHONY: build-bindings
build-bindings:
	$(MAKE) -C ../../bindings build
.PHONY: get-deps-api
get-deps-api: $(ts_bindings_dest)/done.stamp
	# WARN this has a dependency on build-bindings which isn't linked here.

$(ts_bindings_dest)/done.stamp: $(ts_bindings)
	rm -rf $(ts_bindings_dest)
	cp -r $(ts_bindings_src) $(ts_bindings_dest)
	touch $@

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
build: $(ts_bindings_dest)/done.stamp
	$(MAKE) build/done.stamp

build/done.stamp: $(source_files) node_modules/done.stamp tsconfig.json config-overrides.js
	npm run build
	touch $@

.PHONY: build-storybook
build-storybook:
	npm run build-storybook

.PHONY: clean
clean:
	rm -rf build build-storybook node_modules/ src/services/api-ts-sdk

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
