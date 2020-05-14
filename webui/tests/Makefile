.PHONY: get-deps clean e2e-tests docker-e2e-tests

export INTEGRATIONS_HOST_PORT ?= 5300
export DISCREET_LOGS ?= 0

get-deps:
	for i in `seq 3`; do \
		npm ci --no-audit && break ; \
	done

clean-test-artifacts:
	rm -rf cypress/screenshots/*
	rm -rf cypress/videos/*

clean: clean-test-artifacts
	rm -rf node_modules/

test: docker-e2e-tests

test-quiet: docker-e2e-tests-quiet

check:
	yarn run lint

fmt:
	yarn run lint --fix

pre-e2e-tests:
	python ./bin/e2e-tests.py pre-e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT) --discreet-logs $(DISCREET_LOGS)

post-e2e-tests:
	python ./bin/e2e-tests.py post-e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT) --discreet-logs $(DISCREET_LOGS)

run-e2e-tests:
	python ./bin/e2e-tests.py run-e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT) --discreet-logs $(DISCREET_LOGS)

e2e-tests:
	python ./bin/e2e-tests.py e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT) --discreet-logs $(DISCREET_LOGS)

docker-run-e2e-tests:
	python ./bin/e2e-tests.py docker-run-e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT) --discreet-logs $(DISCREET_LOGS)

docker-e2e-tests:
	python ./bin/e2e-tests.py docker-e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT) --discreet-logs $(DISCREET_LOGS)

docker-e2e-tests-quiet:
	python ./bin/e2e-tests.py docker-e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT) --discreet-logs $(DISCREET_LOGS)
