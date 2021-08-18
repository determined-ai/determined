VERSION := $(shell cat ../VERSION)
SHORT_GIT_HASH := $(shell git rev-parse --short HEAD)

ARTIFACTS_DIR := /tmp/artifacts

# Model-hub library environments will be built on top of the default GPU and CPU images in master/pkg/model/defaults.go
DEFAULT_GPU_IMAGE := determinedai/environments:cuda-11.1-pytorch-1.9-lightning-1.3-tf-2.4-gpu-2409e48
# Transformers Args
TRANSFORMERS_VERSION := 4.8.2
DATASETS_VERSION := 1.9.0
TRANSFORMERS_ENVIRONMENT_ROOT := determinedai/model-hub-transformers
# MMDetection Args
# Remember to bump versions in tests/requirements.txt to match.
MMCV_CUDA_VERSION := cu111 # Needs to match DEFAULT_GPU_IMAGE
TORCH_VERSION := 1.9.0 # Needs to match DEFAULT_GPU_IMAGE
MMCV_VERSION := 1.3.9
MMDETECTION_VERSION := 2.14.0
MMDETECTION_ENVIRONMENT_ROOT := determinedai/model-hub-mmdetection

# Example dirs
HF_EXAMPLES := $(wildcard examples/huggingface/*/.)
HF_EXAMPLES_DIRS := $(patsubst examples/huggingface/%/., build-examples/%.tgz, $(HF_EXAMPLES))

.PHONY: clean
clean:
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	rm -rf *.egg-info/
	rm -rf pip-wheel-metadata/
	rm -rf dist/
	rm -rf build/
	rm -rf build-examples/
	find . \( -name __pycache__ -o -name \*.pyc \) -delete
	find . -type d -name .mypy_cache -prune -exec rm -rf {} \;

.PHONY: build
build:
	python setup.py -q bdist_wheel

.PHONY: examples
examples: $(HF_EXAMPLES_DIRS)

build-examples/%.tgz: examples/*/%/
	find "$<" \( -name __pycache__ -o -name \*.pyc \) -delete
	mkdir -p $$(dirname "$@")
	tar -czf "$@" -C $$(dirname "$<") $$(basename "$<")

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
		--build-arg MODEL_HUB_VERSION=$(VERSION) \
		-t $(TRANSFORMERS_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) \
		.

.PHONY: publish-transformers-dev
publish-transformers-dev:
	./docker/publish-docker.sh transformers-gpu-hash $(TRANSFORMERS_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) $(ARTIFACTS_DIR)

.PHONY: build-mmdetection-dev
build-mmdetection-dev:
	docker build -f docker/Dockerfile.mmdetection \
		--build-arg BASE_IMAGE=$(DEFAULT_GPU_IMAGE) \
		--build-arg MMCV_CUDA_VERSION=$(MMCV_CUDA_VERSION) \
		--build-arg TORCH_VERSION=$(TORCH_VERSION) \
		--build-arg MMCV_VERSION=$(MMCV_VERSION) \
		--build-arg MMDETECTION_VERSION=$(MMDETECTION_VERSION) \
		--build-arg MODEL_HUB_VERSION=$(VERSION) \
		-t $(MMDETECTION_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) \
		.

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
	docker build -f docker/Dockerfile.mmdetection \
		--build-arg BASE_IMAGE=$(DEFAULT_GPU_IMAGE) \
		--build-arg MMCV_CUDA_VERSION=$(MMCV_CUDA_VERSION) \
		--build-arg TORCH_VERSION=$(TORCH_VERSION) \
		--build-arg MMCV_VERSION=$(MMCV_VERSION) \
		--build-arg MMDETECTION_VERSION=$(MMDETECTION_VERSION) \
		--build-arg MODEL_HUB_VERSION=$(VERSION) \
		-t $(MMDETECTION_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) \
		-t $(MMDETECTION_ENVIRONMENT_ROOT):$(VERSION) \
		.

.PHONY: publish-mmdetection
publish-mmdetection:
	./docker/publish-docker.sh mmdetection-gpu-hash $(MMDETECTION_ENVIRONMENT_ROOT):$(SHORT_GIT_HASH) $(ARTIFACTS_DIR)
	./docker/publish-docker.sh mmdetection-gpu-version $(MMDETECTION_ENVIRONMENT_ROOT):$(VERSION) $(ARTIFACTS_DIR)

.PHONY: build-docker
build-docker: build-transformers build-mmdetection

.PHONY: publish-docker
publish-docker: publish-transformers publish-mmdetection


