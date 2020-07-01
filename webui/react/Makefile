swagger_client = "src/services/swagger"
PROJECT_ROOT := "../../"

.PHONY: get-deps
get-deps:
	# WARN this module also depends on the swagger generated api client
	# which is not built or updated here. To set up this dependency checkout
	# the swagger generated api client module at ../api-ts-sdk
	for i in `seq 3`; do \
		npm ci --unsafe-perm --no-audit && break ; \
	done

.PHONY: build
build:
	npm run build

.PHONY: build-storybook
build-storybook:
	npm run build-storybook

.PHONY: clean
clean:
	rm -rf build build-storybook node_modules/

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
test:
	npm run test -- --watchAll=false
