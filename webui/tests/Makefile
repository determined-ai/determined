.PHONY: get-deps clean e2e-tests

get-deps:
	for i in `seq 3`; do \
		npm ci --no-audit && break ; \
	done

clean-test-artifacts:
	rm -rf cypress/screenshots/*
	rm -rf cypress/videos/*

clean: clean-test-artifacts
	rm -rf node_modules/

test: e2e-tests

check:
	yarn run lint

fmt:
	yarn run lint --fix

pre-e2e-tests:
	python ./bin/e2e-tests.py pre-e2e-tests

post-e2e-tests:
	python ./bin/e2e-tests.py post-e2e-tests

run-e2e-tests:
	python ./bin/e2e-tests.py run-e2e-tests

e2e-tests:
	python ./bin/e2e-tests.py e2e-tests
