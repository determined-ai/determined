.PHONY: clean
clean:
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	rm -rf *.egg-info/
	rm -rf pip-wheel-metadata/
	rm -rf dist/
	rm -rf build/
	find . \( -name __pycache__ -o -name \*.pyc \) -delete

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


VERSION := $(shell cat ../VERSION)
VERSION_DASHES := $(subst .,-,$(VERSION))
GIT_HASH := $(shell git rev-parse HEAD)
SHORT_GIT_HASH := $(shell git rev-parse --short HEAD)

ARTIFACTS_DIR := /tmp/artifacts

# Model-hub library environments will be built on top of the default GPU and CPU images in master/pkg/model/defaults.go
DEFAULT_GPU_TAG := cuda-10.0-pytorch-1.4-tf-1.15-gpu-067db2b
TRANSFORMERS_VERSION := 4.2.2
DATASETS_VERSION := 1.2.1
export GPU_TRANSFORMERS_ENVIRONMENT_NAME := model-hub-transformers:$(DEFAULT_GPU_TAG)-transformers-$(TRANSFORMERS_VERSION)-datasets-$(DATASETS_VERSION)

.PHONY: build-transformers
build-transformers:
	docker build -f docker/Dockerfile.transformers \
		--build-arg BASE_IMAGE=determinedai/environments:$(DEFAULT_GPU_TAG) \
		--build-arg TRANSFORMERS_VERSION=$(TRANSFORMERS_VERSION) \
		--build-arg DATASETS_VERSION=$(DATASETS_VERSION) \
		--build-arg VERSION=$(VERSION) \
		-t determinedai/$(GPU_TRANSFORMERS_ENVIRONMENT_NAME)-$(GIT_HASH) \
		-t determinedai/$(GPU_TRANSFORMERS_ENVIRONMENT_NAME)-$(SHORT_GIT_HASH) \
		-t determinedai/$(GPU_TRANSFORMERS_ENVIRONMENT_NAME)-$(VERSION) \
		.

.PHONY: publish-transformers
publish-transformers:
	sh docker/publish-docker.sh transformers-gpu determinedai/$(GPU_TRANSFORMERS_ENVIRONMENT_NAME) $(SHORT_GIT_HASH) $(VERSION) $(ARTIFACTS_DIR)


.PHONY: build-docker
build-docker: build-transformers

.PHONY: publish-docker
publish-docker: publish-transformers

.PHONY: update-config-envs
update-config-envs:
	./docker/update-experiment-config-envs.sh $(GPU_TRANSFORMERS_ENVIRONMENT_NAME)-$(SHORT_GIT_HASH)


