TEST_RESULTS_DIR=/tmp/test-results
py_bindings_dest=determined/common/api/bindings.py
cuda_available=$(shell python -c "import torch; print(torch.cuda.is_available())") \

.PHONY: build
build:
	PYTHONWARNINGS=ignore:Normalizing:UserWarning:setuptools.dist \
		python -m build -nxw >/dev/null

.PHONY: publish
publish:
	twine upload --verbose --non-interactive dist/*

.PHONY: publish-ee
publish-ee:
	twine upload --verbose --non-interactive dist/* --repository-url https://push.fury.io/determined-ai

.PHONY: fmt
fmt:
	isort .
	black . --exclude $(py_bindings_dest)

.PHONY: check
check: check-gen
	isort . --check-only
	black . --exclude $(py_bindings_dest) --check
	# Note: plain `flake8` command does not add current directory to sys.path, which causes the
	# flake8-import-restrictions plugin to fail to import the code.
	# https://github.com/atollk/flake8-import-restrictions/issues/13
	python -m flake8 -v
	mypy .


.PHONY: test-cpu
test-cpu:
	mkdir -p $(TEST_RESULTS_DIR)
	coverage run -m pytest -v --runslow --durations=0 -m "not gpu or cpu" --junitxml=$(TEST_RESULTS_DIR)/test-cpu.xml tests

.PHONY: test-gpu-tf
test-gpu-tf:
	mkdir -p $(TEST_RESULTS_DIR)
	coverage run -m pytest -v --runslow --durations=0 -m "gpu" --junitxml=$(TEST_RESULTS_DIR)/test-gpu.xml tests/experiment/keras

.PHONY: test-gpu-deepspeed
test-gpu-deepspeed:
	mkdir -p $(TEST_RESULTS_DIR)
	coverage run -m pytest -v --runslow --durations=0 -m "deepspeed and gpu" --junitxml=$(TEST_RESULTS_DIR)/test-gpu-deepspeed.xml tests/experiment/integrations/test_deepspeed_trial.py

.PHONY: test-gpu-parallel
test-gpu-parallel:
	mkdir -p $(TEST_RESULTS_DIR)
	coverage run -m pytest -v --runslow --durations=0 -m tests/experiment -m "gpu_parallel" --junitxml=$(TEST_RESULTS_DIR)/test-gpu-parallel.xml tests/experiment/pytorch

.PHONY: test-pytorch-cpu
test-pytorch-cpu:
	mkdir -p $(TEST_RESULTS_DIR)
	coverage run -m pytest -v --runslow --durations=0 -m tests/experiment -m "not gpu or cpu" --junitxml=$(TEST_RESULTS_DIR)/test-pytorch-cpu.xml tests/experiment/pytorch

.PHONY: test-pytorch-gpu
test-pytorch-gpu:
	mkdir -p $(TEST_RESULTS_DIR)
	coverage run -m pytest -v --runslow --durations=0 -m tests/experiment -m "gpu" --junitxml=$(TEST_RESULTS_DIR)/test-pytorch-gpu.xml tests/experiment/pytorch

.PHONY: test
test:
	mkdir -p $(TEST_RESULTS_DIR)
	coverage run -m pytest -v --runslow --durations=0 --junitxml=$(TEST_RESULTS_DIR)/test.xml tests

.PHONY: test-tf2
test-tf2:
	mkdir -p $(TEST_RESULTS_DIR)
	coverage run -a -m pytest -v --runslow --durations=0 -m "not gpu" --junitxml=$(TEST_RESULTS_DIR)/test-tf2.xml tests/experiment/keras

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

AWS_TEMPLATES_PATH := determined/deploy/aws/templates

.PHONY: upload-try-now-template
upload-try-now-template: TRY_NOW_TEMPLATE = simple.yaml
upload-try-now-template: TRY_NOW_URL := s3://determined-ai-public/$(TRY_NOW_TEMPLATE)
upload-try-now-template: TEMPLATE_PATH := $(AWS_TEMPLATES_PATH)/$(TRY_NOW_TEMPLATE)
upload-try-now-template:
	aws s3 cp $(TEMPLATE_PATH) $(TRY_NOW_URL) --acl public-read

.PHONY: check-gen
check-gen: aws-master-config-inject
	# Checking that committed, generated code is up-to-date by ensuring that
	# git reports the files as unchanged after forcibly regenerating the files:
	test -z "$(shell git status --porcelain $(AWS_TEMPLATES_PATH))"

.PHONY: gen-deploy-aws-vcpu-mapping
gen-deploy-aws-vcpu-mapping:
	python -m determined.deploy.aws.gen_vcpu_mapping determined/deploy/aws/vcpu_mapping.yaml

.PHONY: aws-master-config-inject
aws-master-config-inject:
	python -m determined.deploy.aws.master_config_inject

.PHONY: install
install:
	pip install .
