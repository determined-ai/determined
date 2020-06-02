.PHONY: get-deps
get-deps:
	for i in `seq 3`; do \
		npm ci --no-audit && break ; \
	done

.PHONY: build
build: build-react build-storybook

.PHONY: build-react
build-react:
	npm run build

.PHONY: build-storybook
build-storybook:
	npm run build-storybook

.PHONY: clean
clean:
	rm -rf build node_modules/

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
