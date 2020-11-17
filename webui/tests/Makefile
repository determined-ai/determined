.PHONY: get-deps
get-deps:
	for i in `seq 3`; do \
		npm ci --no-audit && break ; \
	done

.PHONY: clean-test-artifacts
clean-test-artifacts:
	rm -rf reports/*
	rm -rf logs/*

.PHONY: clean
clean: clean-test-artifacts
	rm -rf node_modules/

.PHONY: test
test:
	python ./bin/e2e-tests.py e2e-tests

.PHONY: check
check:
	npm run lint

.PHONY: fmt
fmt:
	npm run lint --fix

.PHONY: dev-tests
dev-tests:
	python ./bin/e2e-tests.py dev-tests
