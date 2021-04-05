VERSION := $(shell cat ../VERSION)
SHORT_GIT_HASH := $(shell git rev-parse --short HEAD)

ARTIFACTS_DIR := /tmp/artifacts

# Model-hub library environments will be built on top of the default GPU and CPU images in master/pkg/model/defaults.go
DEFAULT_GPU_IMAGE := determinedai/environments:cuda-10.2-pytorch-1.7-tf-1.15-gpu-da9ba40
TRANSFORMERS_VERSION := 4.4.2
DATASETS_VERSION := 1.5.0
TRANSFORMERS_ENVIRONMENT_ROOT := determinedai/model-hub-transformers

.PHONY: clean
clean:
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	rm -rf *.egg-info/
	rm -rf pip-wheel-metadata/
	rm -rf dist/
	rm -rf build/
	find . \( -name __pycache__ -o -name \*.pyc \) -delete
	find . -type d -name .mypy_cache -prune -exec rm -rf {} \;

.PHONY: build
build:
	python setup.py -q bdist_wheel

.PHONY: publish
publish:
	twine upload --verbose --non-interactive dist/*

.PHONY: fmt
fmt:
	isort -y
	black .

.PHONY: check
check:
	isort --check-only
	black . --check
	flake8
	mypy .

.PHONY: test
test:
	pytest -v -s --durations=0 tests

# We will only use HASH for dev environments.
.PHONY: build-transformers-dev
build-transformers-dev:
	docker build -f docker/Dockerfile.transformers \
		--build-arg BASE_IMAGE=$(DEFAULT_GPU_IMAGE) \
		--build-arg TRANSFORMERS_VERSION=$(TRANSFORMERS_VERSION) \
		--build-arg DATASETS_VERSION=$(DATASETS_VERSION) \
		-t $(TRANSFORMERS_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) \
		.

.PHONY: publish-transformers-dev
publish-transformers-dev:
	./docker/publish-docker.sh transformers-gpu $(TRANSFORMERS_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) $(ARTIFACTS_DIR)

.PHONY: build-docker-dev
build-docker-dev: build-transformers-dev

.PHONY: publish-docker-dev
publish-docker-dev: publish-transformers-dev

# We will use a tag of the format determinedai/model-hub-transformers:VERSION for
# master and releases.
.PHONY: build-transformers
build-transformers:
	docker build -f docker/Dockerfile.transformers \
		--build-arg BASE_IMAGE=$(DEFAULT_GPU_IMAGE) \
		--build-arg TRANSFORMERS_VERSION=$(TRANSFORMERS_VERSION) \
		--build-arg DATASETS_VERSION=$(DATASETS_VERSION) \
		-t $(TRANSFORMERS_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) \
		-t $(TRANSFORMERS_ENVIRONMENT_ROOT):$(VERSION) \
		.

.PHONY: publish-transformers
publish-transformers:
	./docker/publish-docker.sh transformers-gpu $(TRANSFORMERS_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) $(ARTIFACTS_DIR)
	./docker/publish-docker.sh transformers-gpu $(TRANSFORMERS_ENVIRONMENT_ROOT):$(VERSION) $(ARTIFACTS_DIR)

.PHONY: build-docker
build-docker: build-transformers

.PHONY: publish-docker
publish-docker: publish-transformers

