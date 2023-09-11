SHELL := /bin/bash
VERSION := $(shell cat ../VERSION)
SHORT_GIT_HASH := $(shell git rev-parse --short HEAD)

ARTIFACTS_DIR := /tmp/artifacts

# Model-hub library environments will be built on top of the default GPU and CPU images in master/pkg/model/defaults.go
DEFAULT_GPU_IMAGE := determinedai/environments:cuda-11.3-pytorch-1.12-tf-2.11-gpu-2b7e2a1

############REMINDER############
# When bumping third-party library versions, remember to bump versions in
# tests/requirements.txt and docs/requirements.txt to match.
############REMINDER############

# Transformers Args
TRANSFORMERS_VERSION := 4.8.2
DATASETS_VERSION := 1.9.0
TRANSFORMERS_ENVIRONMENT_ROOT := determinedai/model-hub-transformers
# MMDetection Args
MMCV_CUDA_VERSION := cu113 # Needs to match DEFAULT_GPU_IMAGE
TORCH_VERSION := 1.12.0 # Needs to match DEFAULT_GPU_IMAGE
MMCV_VERSION := 1.7.1 # Needs to support the above Torch version
MMDETECTION_VERSION := 2.27.0
MMDETECTION_ENVIRONMENT_ROOT := determinedai/model-hub-mmdetection

.PHONY: clean
clean:
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	rm -rf *.egg-info/
	rm -rf pip-wheel-metadata/
	rm -rf dist/
	rm -rf build/
	rm -f .coverage
	find . \( -name __pycache__ -o -name \*.pyc \) -delete
	find . -type d -name .mypy_cache -prune -exec rm -rf {} \;

.PHONY: build
build:
	PYTHONWARNINGS=ignore:Normalizing:UserWarning:setuptools.dist \
		python -m build -nxw >/dev/null

.PHONY: publish
publish:
	twine upload --verbose --non-interactive dist/*

.PHONY: fmt
fmt:
	isort .
	black .

.PHONY: check
check:
	isort . --check-only
	black . --check
	python -m flake8
	mypy .

.PHONY: test
test:
	coverage run -m pytest -v -s --durations=0 tests

# We will only use HASH for dev environments.
.PHONY: build-transformers-dev
build-transformers-dev:
	docker build -f docker/Dockerfile.transformers \
		--build-arg BASE_IMAGE=$(DEFAULT_GPU_IMAGE) \
		--build-arg TRANSFORMERS_VERSION=$(TRANSFORMERS_VERSION) \
		--build-arg DATASETS_VERSION=$(DATASETS_VERSION) \
		--build-arg MODEL_HUB_VERSION=$(VERSION) \
		-t $(TRANSFORMERS_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) \
		.

.PHONY: publish-transformers-dev
publish-transformers-dev:
	./docker/publish-docker.sh transformers-gpu-hash $(TRANSFORMERS_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) $(ARTIFACTS_DIR)

.PHONY: build-mmdetection-dev
build-mmdetection-dev:
	if [[ "$(DEFAULT_GPU_IMAGE)" != *"cuda-11.3"* ]]; then \
	  echo "Expected $(DEFAULT_GPU_IMAGE) to have cuda version $(MMCV_CUDA_VERSION)."; \
	  exit 1; \
	elif [[ "$(DEFAULT_GPU_IMAGE)" != *"pytorch-1.12"* ]]; then \
	  echo "Expected $(DEFAULT_GPU_IMAGE) to have pytorch version $(TORCH_VERSION)."; \
	  exit 1; \
	else \
	  docker build -f docker/Dockerfile.mmdetection \
	  	  --build-arg BASE_IMAGE=$(DEFAULT_GPU_IMAGE) \
	  	  --build-arg MMCV_CUDA_VERSION=$(MMCV_CUDA_VERSION) \
	  	  --build-arg TORCH_VERSION=$(TORCH_VERSION) \
	  	  --build-arg MMCV_VERSION=$(MMCV_VERSION) \
	  	  --build-arg MMDETECTION_VERSION=$(MMDETECTION_VERSION) \
	  	  --build-arg MODEL_HUB_VERSION=$(VERSION) \
	  	  -t $(MMDETECTION_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) \
	  	  .; \
	fi;

.PHONY: publish-mmdetection-dev
publish-mmdetection-dev:
	./docker/publish-docker.sh mmdetection-gpu-hash $(MMDETECTION_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) $(ARTIFACTS_DIR)

.PHONY: build-docker-dev
build-docker-dev: build-transformers-dev build-mmdetection-dev

.PHONY: publish-docker-dev
publish-docker-dev: publish-transformers-dev publish-mmdetection-dev

# We will use a tag of the format determinedai/model-hub-transformers:VERSION for
# master and releases.
.PHONY: build-transformers
build-transformers:
	docker build -f docker/Dockerfile.transformers \
		--build-arg BASE_IMAGE=$(DEFAULT_GPU_IMAGE) \
		--build-arg TRANSFORMERS_VERSION=$(TRANSFORMERS_VERSION) \
		--build-arg DATASETS_VERSION=$(DATASETS_VERSION) \
		--build-arg MODEL_HUB_VERSION=$(VERSION) \
		-t $(TRANSFORMERS_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) \
		-t $(TRANSFORMERS_ENVIRONMENT_ROOT):$(VERSION) \
		.

.PHONY: publish-transformers
publish-transformers:
	./docker/publish-docker.sh transformers-gpu-hash $(TRANSFORMERS_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) $(ARTIFACTS_DIR)
	./docker/publish-docker.sh transformers-gpu-version $(TRANSFORMERS_ENVIRONMENT_ROOT):$(VERSION) $(ARTIFACTS_DIR)

.PHONY: build-mmdetection
build-mmdetection:
	if [[ "$(DEFAULT_GPU_IMAGE)" != *"cuda-11.3"* ]]; then \
	  echo "Expected $(DEFAULT_GPU_IMAGE) to have cuda version $(MMCV_CUDA_VERSION)."; \
	  exit 1; \
	elif [[ "$(DEFAULT_GPU_IMAGE)" != *"pytorch-1.12"* ]]; then \
	  echo "Expected $(DEFAULT_GPU_IMAGE) to have pytorch version $(TORCH_VERSION)."; \
	  exit 1; \
	else \
	  docker build -f docker/Dockerfile.mmdetection \
		  --build-arg BASE_IMAGE=$(DEFAULT_GPU_IMAGE) \
		  --build-arg MMCV_CUDA_VERSION=$(MMCV_CUDA_VERSION) \
		  --build-arg TORCH_VERSION=$(TORCH_VERSION) \
		  --build-arg MMCV_VERSION=$(MMCV_VERSION) \
		  --build-arg MMDETECTION_VERSION=$(MMDETECTION_VERSION) \
		  --build-arg MODEL_HUB_VERSION=$(VERSION) \
		  -t $(MMDETECTION_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) \
		  -t $(MMDETECTION_ENVIRONMENT_ROOT):$(VERSION) \
		  .; \
	fi;

.PHONY: publish-mmdetection
publish-mmdetection:
	./docker/publish-docker.sh mmdetection-gpu-hash $(MMDETECTION_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) $(ARTIFACTS_DIR)
	./docker/publish-docker.sh mmdetection-gpu-version $(MMDETECTION_ENVIRONMENT_ROOT):$(VERSION) $(ARTIFACTS_DIR)

.PHONY: build-docker
build-docker: build-transformers build-mmdetection

.PHONY: publish-docker
publish-docker: publish-transformers publish-mmdetection
