py_bindings_dest=determined/common/api/bindings.py
cuda_available=$(shell python -c "import torch; print(torch.cuda.is_available())") \

.PHONY: build
build: gen
	python -W ignore:Normalizing:UserWarning:setuptools.dist setup.py -q bdist_wheel

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
	flake8
	mypy .

.PHONY: test-cpu
test-cpu: gen
	coverage run -m pytest -v --runslow --durations=0 -m "not gpu" tests

.PHONY: test-gpu
test-gpu: gen
	# We will use this target in circleci once we have access to gpu instances.
	coverage run -m pytest -v --runslow --durations=0 -m "gpu" tests

.PHONY: test
test: gen
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
	coverage run -a -m pytest -v --runslow --durations=0 tests/experiment/keras

.PHONY: clean
clean: ungen
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

GENERATED_COMMITTED := determined/common/schemas/expconf/_gen.py
GENERATION_INPUTS = ../schemas/gen.py $(shell find ../schemas/expconf -name '*.json')

.PHONY: ungen
ungen:
	git checkout -- $(GENERATED_COMMITTED)

.PHONY: gen
gen: $(GENERATED_COMMITTED)

.PHONY: force-gen
force-gen:
	touch ../schemas/gen.py

.PHONY: check-gen
check-gen: force-gen aws-master-config-inject $(GENERATED_COMMITTED)
	# Checking that committed, generated code is up-to-date by ensuring that
	# git reports the files as unchanged after forcibly regenerating the files:
	test -z "$(shell git status --porcelain $(GENERATED_COMMITTED))"
	test -z "$(shell git status --porcelain $(AWS_TEMPLATES_PATH))"

determined/common/schemas/expconf/_gen.py: $(GENERATION_INPUTS)
	../schemas/gen.py python\
		--package expconf \
		--output $@

.PHONY: gen-deploy-aws-vcpu-mapping
gen-deploy-aws-vcpu-mapping:
	python -m determined.deploy.aws.gen_vcpu_mapping determined/deploy/aws/vcpu_mapping.yaml

.PHONY: aws-master-config-inject
aws-master-config-inject:
	python -m determined.deploy.aws.master_config_inject

.PHONY: install
install:
	pip install .
