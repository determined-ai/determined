swagger_client = "src/services/swagger"
PROJECT_ROOT := "../../"

.PHONY: get-deps
get-deps:
	for i in `seq 3`; do \
		npm ci --no-audit && break ; \
	done

.PHONY: get-api
get-api:
	# WARN this module also depends on the swagger generated api client
	# which is not built or updated here. To set up this dependency checkout
	# the swagger generated api client module at ../api-ts-sdk
	cp -r ../api-ts-sdk/src src/services/api-ts-sdk

.PHONY: build
build: get-api
	npm run build

.PHONY: build-storybook
build-storybook: get-api
	npm run build-storybook

.PHONY: clean
clean:
	rm -rf build build-storybook node_modules/ src/services/api-ts-sdk

.PHONY: live
live:
	npm start

.PHONY: check
check:
	npm run lint

.PHONY: fmt
fmt:
	npm run lint:js -- --fix
	npm run lint:css -- --fix

.PHONY: test
test: get-api
	npm run test -- --watchAll=false
