.PHONY: get-deps clean e2e-tests docker-e2e-tests

export INTEGRATIONS_HOST_PORT ?= 5300

get-deps:
	for i in `seq 3`; do \
		npm ci --no-audit && break ; \
	done

clean:
	rm -rf cypress/screenshots/*
	rm -rf cypress/videos/*

clean-deps:
	rm -rf node_modules/

test: docker-e2e-tests

check:
	yarn run lint

fmt:
	yarn run lint --fix

pre-e2e-tests:
	python ./bin/e2e-tests.py pre-e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT)

post-e2e-tests:
	python ./bin/e2e-tests.py post-e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT)

run-e2e-tests:
	python ./bin/e2e-tests.py run-e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT)

e2e-tests:
	python ./bin/e2e-tests.py e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT)

docker-run-e2e-tests:
	python ./bin/e2e-tests.py docker-run-e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT)

docker-e2e-tests:
	python ./bin/e2e-tests.py docker-e2e-tests --integrations-host-port $(INTEGRATIONS_HOST_PORT)
