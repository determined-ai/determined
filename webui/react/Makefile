# Set the path React is going to be served from eg ./ for serving from root.
export PUBLIC_URL ?= /det
export ESLINT_NO_DEV_ERRORS ?= true
export NODE_OPTIONS=--max-old-space-size=6144
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
	git submodule update --init

.PHONY: build
build: build/done.stamp

build/done.stamp: $(source_files) node_modules/done.stamp tsconfig.json craco.config.js jest.config.js vite.config.ts
	npm run build
	touch $@

.PHONY: clean
clean:
	rm -rf build build-storybook node_modules/

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
