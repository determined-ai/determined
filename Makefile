.PHONY: \
  all \
  build \
  build-agent \
  build-agent-docker \
  build-cloud-images \
  build-docker \
  build-master \
  build-master-docker \
  bump-version \
  check \
  check-fmt \
  check-types \
  check-python-assert \
  clean \
  fmt \
  get-deps \
  graphql \
  graphql-elm \
  graphql-python \
  graphql-schema \
  pin-deps \
  upgrade-deps \
  publish \
  test \
  test-all \
  test-cloud-integrations \
  test-python-integrations \
  test-integrations \
  webui

export VERSION := $(shell cat VERSION)
export INTEGRATIONS_HOST_PORT ?= 8080
export DB_HOST_PORT ?= 5433
export INTEGRATIONS_LOG_OPTS := --no-color

GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_DIRTY := $(if $(shell git status --porcelain),-dirty,)
export DET_GIT_COMMIT := $(GIT_COMMIT)$(GIT_DIRTY)

export GO111MODULE := on
GOBIN ?= $(shell go env GOPATH)/bin
GORELEASER_VERSION := v0.128.0

BUILDDIR ?= build

MYPY := mypy
TYPE_CHECK_PATHS := CI/integrations tests webui/elm/pytests

# This can be given as a prefix to a command to run that command with all staged
# and committed Python files in the repo as arguments.
RUN_ON_PYTHON_PATHS := git ls-files -z '*.py' | xargs -0

# Ignoring examples because isort does not play well with packages that are not
# in the virtualenv.
ISORT_RUN_ON_PYTHON_PATHS := git ls-files -z '*.py' ':!:*/__init__.py' ':!:examples' | xargs -0

FLAKE_RUN_ON_PYTHON_PATHS := git ls-files -z \
	'*.py' \
	':!:examples/experimental/FasterRCNN_tp/*' \
	':!:examples/experimental/resnet50_tf_keras/tensorflow_files/*' \
	':!:examples/experimental/bert_glue_pytorch/download_glue_data.py' \
	':!:examples/experimental/nas_search/randomNAS_files/*' \
	':!:examples/tutorials/native-tf-keras/*' \
	| xargs -0

export DOCKER_REGISTRY ?=
DET_DEV_AGENT_IMAGE := determinedai/determined-dev:determined-agent-$(DET_GIT_COMMIT)
DET_DEV_MASTER_IMAGE := determinedai/determined-dev:determined-master-$(DET_GIT_COMMIT)
export DET_IMAGES := $(DET_DEV_AGENT_IMAGE),$(DET_DEV_MASTER_IMAGE)

# These variables are picked up by GoReleaser for the master build; we default to including no keys.
export DET_SEGMENT_MASTER_KEY ?=
export DET_SEGMENT_WEBUI_KEY ?=

all: get-deps build-docker

# combined-reqs.txt contains the pinned versions for all development
# dependencies in this repo, including transitive dependencies.
REQUIREMENTS_IN := combined-reqs.in
REQUIREMENTS_OUTPUT := combined-reqs.txt

get-deps: python-get-deps go-get-deps
	$(MAKE) WEBUI_TARGET=$@ webui

go-get-deps:
	$(MAKE) -C master get-deps
	$(MAKE) -C agent get-deps
	curl -fsSL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh -s -- -b $(GOBIN) $(GORELEASER_VERSION)
	go get github.com/talos-systems/conform@fa7df19996ece307285da44c73f210c6cbec9207

python-get-deps:
	pip install -r $(REQUIREMENTS_OUTPUT)


pin-deps:
	pip-compile -q --output-file $(REQUIREMENTS_OUTPUT) $(REQUIREMENTS_IN)

upgrade-deps:
	pip-compile -q --upgrade --output-file $(REQUIREMENTS_OUTPUT) $(REQUIREMENTS_IN)

bump-version: PART?=patch
bump-version:
	bumpversion $(PART)

webui:
	$(MAKE) -C webui/elm ${WEBUI_TARGET}
	$(MAKE) -C webui/react ${WEBUI_TARGET}

build: build-master build-agent

build-agent:
	$(MAKE) -C agent build

build-master:
	$(MAKE) -C master build

debs:
	cp -r packaging "$(BUILDDIR)"
	cd "$(BUILDDIR)" && GORELEASER_CURRENT_TAG=$(VERSION) $(GOBIN)/goreleaser -f $(CURDIR)/.goreleaser.yml --snapshot --rm-dist

build-docker: build debs
	$(MAKE) build-master-docker build-agent-docker

build-agent-docker:
	$(MAKE) -C agent build-docker

build-master-docker:
	$(MAKE) -C master build-docker

publish-dev:
	$(MAKE) -C master $@
	$(MAKE) -C agent $@

clean:
	rm -rf build
	find . \( -name __pycache__ -o -name \*.pyc -o -name .mypy_cache \) -print0 | xargs -0 rm -rf
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C docs $@
	$(MAKE) -C common $@
	$(MAKE) -C harness $@
	$(MAKE) -C cli $@
	$(MAKE) -C deploy $@
	$(MAKE) WEBUI_TARGET=$@ webui

guard-publish:
	@if [ -n "$(GIT_DIRTY)" ]; then \
		echo "You cannot publish with a dirty git working tree."; exit 1; fi
	@if [ "$$(git tag --points-at HEAD)" != "v$(VERSION)" ]; then \
		echo "Ensure that the tag v$(VERSION) (and no other tag) points to the current commit."; exit 1; fi
	@if ! command -v twine >/dev/null 2>&1; then \
		echo "You must have twine installed."; exit 1; fi
	@if ! [ \( -n "$$TWINE_USERNAME" -a -n "$$TWINE_PASSWORD" \) -o -f ~/.pypirc ]; then \
		echo "You must set the TWINE_USERNAME and TWINE_PASSWORD environment variables or set up a ~/.pypirc file."; exit 1; fi

# Publish release artifacts. See RELEASE.md for dependencies (awscli,
# terraform, etc.) and details.
#
# For safety's sake, we make a best-effort attempt to avoid overwriting
# existing objects. If you intend to overwrite an existing package, just
# remove the current object from the S3 bucket manually and retry.
publish: guard-publish clean all
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C common $@
	$(MAKE) -C harness $@
	$(MAKE) -C cli $@
	$(MAKE) -C deploy $@

	cp -r packaging "$(BUILDDIR)"
	cd "$(BUILDDIR)" && $(GOBIN)/goreleaser -f $(CURDIR)/.goreleaser.yml --rm-dist

	# Upload the docs last because it updates the terraform state file,
	# which dirties the working directory.
	$(MAKE) -C docs $@

# This target assumes that a Hasura instance is running and queries it to
# retrieve the current schema files, producing a schema file that the
# `graphql-elm` and `graphql-python` targets can then use to generate code
# without having to have a server running.
graphql-schema:
	scripts/hasura/export-metadata.sh
	python -m sgqlc.introspection \
		-H "X-Hasura-Admin-Secret: $${DET_HASURA_SECRET:-hasura}" \
		-H "X-Hasura-Role: user" \
		http://localhost:8081/v1/graphql \
		master/graphql-schema.json

graphql-python:
	sgqlc-codegen master/graphql-schema.json common/determined_common/api/gql.py
	black common/determined_common/api/gql.py
	isort common/determined_common/api/gql.py

graphql-elm:
	$(MAKE) -C webui/elm graphql

graphql:
	$(MAKE) graphql-schema
	$(MAKE) graphql-python graphql-elm

check: check-python check-commit-messages
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) WEBUI_TARGET=$@ webui

check-python: check-python-fmt check-python-types check-python-assert

check-python-fmt:
	$(ISORT_RUN_ON_PYTHON_PATHS) isort --check
	$(RUN_ON_PYTHON_PATHS) black --check
	$(FLAKE_RUN_ON_PYTHON_PATHS) flake8

check-python-types:
	$(MYPY) $(TYPE_CHECK_PATHS)
	$(MYPY) cli
	$(MYPY) common
	$(MYPY) harness

check-python-assert:
	@scripts/lint-assert.sh

check-commit-messages:
	$(GOBIN)/conform enforce

fmt:
	$(ISORT_RUN_ON_PYTHON_PATHS) isort
	$(RUN_ON_PYTHON_PATHS) black
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) -C webui $@

# TEST_EXPR can be used to only run tests which match the given substring
# expression, using the pytest "-k" flag.
# Example: `make test-integrations -e TEST_EXPR=warm_start` will only run the
# integration tests with "warm_start" in their name
TEST_EXPR ?= ""
PYTEST_MARKS ?= ""

test:
	pytest -v -k $(TEST_EXPR) \
		tests/unit/ tests/cli/
	$(MAKE) -C master $@
	$(MAKE) -C agent $@
	$(MAKE) WEBUI_TARGET=$@ webui

test-tf2:
	pip freeze | grep "tensorflow==2.*"
	pytest -v -k $(TEST_EXPR) --runslow \
		--durations=0 \
		tests/unit/experiment/tensorflow/test_estimator_trial.py \
		tests/unit/experiment/tensorflow/test_util.py
	# We must run these tests separately becuase estimators need to disable v2
	# behavior (a global operation). We are explicitly testing eager execution
	# for tf keras which needs v2 behavior enabled. You can't enable v2 behavior
	# anywhere but the "start" of your program. See:
	# https://github.com/tensorflow/tensorflow/issues/18304#issuecomment-379435515.
	pytest -v -k $(TEST_EXPR) --runslow \
		--durations=0 \
		tests/unit/experiment/keras/test_tf_keras_trial.py \
		tests/unit/experiment/keras/test_keras_data.py

test-harness:
	pytest -v -k $(TEST_EXPR) --runslow \
		--durations=0 \
		tests/unit

test-python-integrations: MASTER_HOST ?= localhost
test-python-integrations: MASTER_CONFIG_PATH ?=
test-python-integrations:
	@echo "Running integration tests on port $(INTEGRATIONS_HOST_PORT)"
	pytest -vv -s \
		-k "$(TEST_EXPR)" \
		-m "$(PYTEST_MARKS)" \
		--durations=0 \
		--master-host="$(MASTER_HOST)" \
		--master-port="$(INTEGRATIONS_HOST_PORT)" \
		--master-config-path="$(MASTER_CONFIG_PATH)" \
		--junit-xml=build/test-reports/integ-test.xml \
		--capture=fd \
		--require-secrets \
		tests/integrations

test-master-integrations:
	$(MAKE) -C master test-integrations

test-integrations: test-python-integrations test-master-integrations

test-performance:
	pytest -v -s tests/integrations/performance
