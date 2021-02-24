swagger_client = "src/services/swagger"
export PROJECT_ROOT := "../../"
# Set the path React is going to be served from eg ./ for serving from root.
export PUBLIC_URL ?= /det

.PHONY: default
default: get-deps
	$(MAKE) build

.PHONY: get-deps-api
get-deps-api:
	rm -rf src/services/api-ts-sdk
	cp -r ../../api-bindings/typescript-fetch-bindings src/services/api-ts-sdk

.PHONY: get-deps-npm
get-deps-npm:
	for i in `seq 3`; do \
		npm ci --no-audit && break ; \
	done

.PHONY: get-deps
get-deps: get-deps-api get-deps-npm
	# WARN this module also depends on the swagger generated api client
	# which is not built or updated here. To set up this dependency checkout
	# the swagger generated api client module at ../api-ts-sdk

.PHONY: build
build:
	npm run build

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
