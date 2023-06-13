# Set the path React is going to be served from eg ./ for serving from root.
export PUBLIC_URL ?= /det
export ESLINT_NO_DEV_ERRORS ?= true
export NODE_OPTIONS=--max-old-space-size=6144
export PLAYWRIGHT_BROWSERS_PATH=0
source_files := $(shell find src public -type f)
merge_base := $(shell git merge-base main HEAD)

.PHONY: default
default: get-deps
	$(MAKE) build

node_modules/playwright-core/.local-browsers:
	npx playwright install

.PHONY: get-deps-npm
get-deps-npm:
	npm ci --no-audit
	$(MAKE) node_modules/playwright-core/.local-browsers

node_modules/done.stamp: package-lock.json
	$(MAKE) get-deps-npm
	touch $@

.PHONY: get-deps
get-deps: node_modules/done.stamp
	git submodule update --init

.PHONY: build
build: build/done.stamp

build/done.stamp: $(source_files) node_modules/done.stamp tsconfig.json vite.config.ts
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
	VITEST_MAX_THREADS=6 VITEST_MIN_THREADS=1 npm run test:coverage -- --reporter=junit --reporter=default --outputFile.junit=coverage/junit.xml

# INCLUDE_FLAKY runs all tests even if they use flakyIt.
# By default this would run ALL tests.
# To avoid duplication, I use a filename (LogViewer) here to specify which group of tests to run.
.PHONY: test-ci-flaky
test-ci-flaky:
	INCLUDE_FLAKY=true VITEST_MAX_THREADS=6 VITEST_MIN_THREADS=1 npm run test -- LogViewer --reporter=default

.PHONY: visual-test-deps
visual-test-deps: node_modules/playwright-core/.local-browsers
	npx playwright install-deps

screenshots/HEAD: node_modules visual-test-deps
	node scripts/designkit-screenshot.mjs HEAD

screenshots/MAIN: node_modules visual-test-deps
	cp -r node_modules/playwright-core/.local-browsers browsers
	git checkout $(merge_base)
	npm ci --no-audit
	cp -r browsers node_modules/playwright-core/.local-browsers
	node scripts/designkit-screenshot.mjs MAIN
	git reset --hard HEAD
	rm -rf new-scripts
	git checkout -
	npm ci --no-audit

screenshot-summary.html: screenshots/HEAD screenshots/MAIN
	node scripts/make-screenshot-diff.mjs

.PHONY: visual-test
visual-test: screenshot-summary.html
