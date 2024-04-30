# Set the path React is going to be served from eg ./ for serving from root.
export PUBLIC_URL ?= /det
export ESLINT_NO_DEV_ERRORS ?= true
export NODE_OPTIONS=--max-old-space-size=6144
export MOCK_ROOT=src/e2e/mocks
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
get-deps: node_modules/done.stamp

.PHONY: build
build: build/done.stamp

build/done.stamp: $(source_files) node_modules/done.stamp tsconfig.json vite.config.mts
	npm run build
	touch $@

.PHONY: clean
clean:
	rm -rf build node_modules/

.PHONY: live
live: start
.PHONY: start
start:
	npm start

.PHONY: prettier-check
prettier:
	npm run prettier $(PRE_ARGS)
.PHONY: stylelint
stylelint:
	npm run stylelint $(ST_ARGS)
.PHONY: eslint
eslint:
	npm run eslint $(ES_ARGS)

.PHONY: check-eslint
check-eslint:
	npm run lint:eslint
.PHONY: check-stylelint
check-stylelint:
	npm run lint:style
check-prettier-js:
	npm run format-check:js > /dev/null
check-prettier-css:
	npm run format-check:css > /dev/null
check-prettier-misc:
	npm run format-check:misc > /dev/null
.PHONY: check-package-lock
check-package-lock:
	if grep 'ssh://' package-lock.json ; then echo "ssh url in package-lock.json, please convert to https url" ; false ; fi
.PHONY: check-prettier
check-prettier: check-prettier-js check-prettier-css check-prettier-misc
.PHONY: check
check: check-eslint check-stylelint check-prettier check-package-lock


.PHONY: fmt-js-eslint
fmt-js-eslint:
	npm run lint:eslint -- --fix
.PHONY: fmt-js-prettier
fmt-js-prettier:
	npm run format:js > /dev/null
.PHONY: fmt-js
fmt-js: fmt-js-eslint fmt-js-prettier
.PHONY: fmt-css-stylelint
fmt-css-stylelint:
	npm run lint:style -- --fix
.PHONY: fmt-css-prettier
fmt-css-prettier:
	npm run format:css > /dev/null
.PHONY: fmt-css
fmt-css: fmt-css-stylelint fmt-css-prettier
.PHONY: fmt-misc
fmt-misc:
	npm run  format:misc > /dev/null
.PHONY: fmt-package-lock
fmt-package-lock:
# check if we're running on MacOS
ifeq "$(shell uname)" "Darwin"
	sed -i '' -e 's|git+ssh://git@github.com|https://github.com|' package-lock.json
else
	sed -i -e 's|git+ssh://git@github.com|https://github.com|' package-lock.json
endif
.PHONY: fmt
fmt: fmt-js fmt-css fmt-misc fmt-package-lock

.PHONY: test
test:
	npm run test -- run

.PHONY: test-ci
test-ci:
	VITEST_MAX_THREADS=6 VITEST_MIN_THREADS=1 npm run test:coverage -- --reporter=junit --reporter=default --outputFile.junit=junit.xml

# INCLUDE_FLAKY=true runs all tests with @flaky in the test name
.PHONY: test-ci-flaky
test-ci-flaky:
	INCLUDE_FLAKY=true VITEST_MAX_THREADS=6 VITEST_MIN_THREADS=1 npm run test -- --reporter=default

.PHONY: get-playwright
get-playwright:
	npx playwright install

.PHONY: get-playwright-ci
get-playwright-ci:
	npx playwright install --with-deps 
	npx playwright install chrome 
	npx playwright install webkit
	npx playwright install firefox

.PHONY: mb-start
mb-start:
	# npm install mountebank
	npx mb start --allowInjection --localonly >"/tmp/mb-playwright.log" 2>&1 &
	sleep 3

.PHONY: mb-stop
mb-stop:
	npx mb stop

# Downloads and saves to a file the recorded requests from Mountebank
.PHONY: mb-save-imposters
mb-save-imposters:
	@curl -X GET -H 'Content-Type: application/json' \
		"http://localhost:2525/imposters?replayabe=true&removeProxies=true" > "${MOCK_ROOT}/debug-saved-imposters.json"

# Injects a script to MB that records requests and responses to the backend so that they can be re-used with mb-start-saved-imposters
.PHONY: mb-record-imposters
mb-record-imposters: 
	injection_script=$(tr -d '\n' <src/e2e/mocks/predicate-generator.js)
	@curl -i -X PUT -H 'Content-Type: application/json' \
		http://localhost:2525/imposters \
		--data "@${MOCK_ROOT}/proxy-always.json"

# Applies previously saved imposters in mountebank without recording requests. Requests that are not present may
# result in Not Found errors as this does not forward requests to the backend, but instead imposters all requests.
.PHONY: mb-start-saved-imposters
mb-start-saved-imposters:
	@curl -i -X PUT -H 'Content-Type: application/json' \
		http://localhost:2525/imposters \
		--data "@${MOCK_ROOT}/saved-imposters.json"
