py_bindings_dest=determined/common/api/bindings.py
cuda_available=$(shell python -c "import torch; print(torch.cuda.is_available())") \

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
	black . --exclude $(py_bindings_dest)

.PHONY: check
check: check-gen
	isort . --check-only
	black . --exclude $(py_bindings_dest) --check
	# Note: plain `flake8` command does not add current directory to sys.path, which causes the
	# flake8-import-restrictions plugin to fail to import the code.
	# https://github.com/atollk/flake8-import-restrictions/issues/13
	python -m flake8
	mypy .

.PHONY: test-cpu
test-cpu:
	coverage run -m pytest -v --runslow --durations=0 -m "not gpu or cpu" tests

.PHONY: test-cpu-tf1
test-cpu-tf1:
	coverage run -m pytest -v --runslow --durations=0 -m "tf1_cpu" tests/experiment/keras tests/experiment/tensorflow --ignore=tests/experiment/integrations

.PHONY: test-gpu
test-gpu:
	# Since integration tests are image-specific, they must be excluded and run separately
	coverage run -m pytest -v --runslow --durations=0 -m "gpu" tests/experiment --ignore=tests/experiment/integrations

.PHONY: test-gpu-lightning
test-gpu-lightning:
	# Lightning tests require their own docker image in circleci
	coverage run -m pytest -v --runslow --durations=0 -m "pytorch_lightning" tests/experiment/integrations/test_pytorch_lightning.py

.PHONY: test-gpu-deepspeed
test-gpu-deepspeed:
	coverage run -m pytest -v --runslow --durations=0 -m "deepspeed and gpu" tests/experiment/integrations/test_deepspeed_trial.py

.PHONY: test-gpu-parallel
test-gpu-parallel:
	coverage run -m pytest -v --runslow --durations=0 -m tests/experiment -m "gpu_parallel" tests/experiment --ignore=tests/experiment/integrations

.PHONY: test-pytorch-cpu
test-pytorch-cpu:
	coverage run -m pytest -v --runslow --durations=0 -m tests/experiment -m "not gpu or cpu" tests/experiment/pytorch

.PHONY: test-pytorch-gpu
test-pytorch-gpu:
	coverage run -m pytest -v --runslow --durations=0 -m tests/experiment -m "gpu" tests/experiment/pytorch

.PHONY: test
test:
	coverage run -m pytest -v --runslow --durations=0 tests

.PHONY: test-tf2
test-tf2:
	pip freeze | grep "tensorflow==2.*"
	coverage run -m pytest -v --runslow --durations=0 tests/experiment/tensorflow
	# We must run these tests separately becuase estimators need to disable v2
	# behavior (a global operation). We are explicitly testing eager execution
	# for tf keras which needs v2 behavior enabled. You can't enable v2 behavior
	# anywhere but the "start" of your program. See:
	# https://github.com/tensorflow/tensorflow/issues/18304#issuecomment-379435515.
	coverage run -a -m pytest -v --runslow --durations=0 -m "not gpu" tests/experiment/keras

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
