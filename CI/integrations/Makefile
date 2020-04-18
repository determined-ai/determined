.PHONY: build cleanup get-images run-master-integration-tests run-python-integration-tests

ifeq ($(origin DET_GIT_COMMIT), undefined)
GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_DIRTY := $(if $(shell git status --porcelain),-dirty,)
export DET_GIT_COMMIT := $(GIT_COMMIT)$(GIT_DIRTY)
endif

VERSION := $(shell cat ../../VERSION)
INTEGRATIONS_RESOURCE_SUFFIX ?=
TF1_CPU_IMAGE := determinedai/environments:py-3.6.9-pytorch-1.4-tf-1.14-cpu-90bf50b
TF2_CPU_IMAGE := determinedai/environments:py-3.6.9-pytorch-1.4-tf-2.1-cpu-90bf50b
TF1_GPU_IMAGE := determinedai/environments:cuda-10.0-pytorch-1.4-tf-1.14-gpu-90bf50b
TF2_GPU_IMAGE := determinedai/environments:cuda-10.1-pytorch-1.4-tf-2.1-gpu-90bf50b
DET_DEV_AGENT_IMAGE := determinedai/determined-dev:determined-agent-$(DET_GIT_COMMIT)
DET_DEV_MASTER_IMAGE := determinedai/determined-dev:determined-master-$(DET_GIT_COMMIT)
DET_IMAGES := $(DET_DEV_AGENT_IMAGE),$(DET_DEV_MASTER_IMAGE)
DOCKER_REGISTRY ?=
TEST_TAG ?= "$(DOCKER_REGISTRY)determinedai/determined-dev:determined-test-harness-$(DET_GIT_COMMIT)"
TEST_EXPR ?=
PYTEST_MARKS ?=
MASTER_CONFIG_PATH ?=
REPORT_ROOT ?=

build:
	docker build -f Dockerfile -t $(TEST_TAG) ../../

cleanup:
	-docker kill $$(docker ps -q)
	-docker rm $$(docker ps -a -q)
	docker volume prune -f
	docker network prune -f

get-images:
	@python ci_docker_images.py --registry "$(DOCKER_REGISTRY)" --version "$(VERSION)" "$(DET_IMAGES)"

pull-environments-images:
	docker pull $(TF1_CPU_IMAGE)
	docker pull $(TF2_CPU_IMAGE)
	docker pull $(TF1_GPU_IMAGE)
	docker pull $(TF2_GPU_IMAGE)

publish-dev:
	# Show dirty files, if there are any.
	git status --porcelain
	docker push $(TEST_TAG)

run-master-integration-tests:
	docker run --rm $(TEST_TAG) make -C master test-integrations

run-python-integration-tests:
	docker run --rm \
		-e INTEGRATIONS_RESOURCE_SUFFIX=$(INTEGRATIONS_RESOURCE_SUFFIX) \
		-e TEST_EXPR="$(TEST_EXPR)" \
		-e PYTEST_MARKS="$(PYTEST_MARKS)" \
		-e MASTER_CONFIG_PATH="$(MASTER_CONFIG_PATH)" \
		-v $(REPORT_ROOT):/build \
		-v /tmp:/tmp \
		-v $(MASTER_CONFIG_PATH):$(MASTER_CONFIG_PATH) \
		--network host \
		-v /var/run/docker.sock:/var/run/docker.sock \
		$(TEST_TAG) \
		make test-python-integrations
