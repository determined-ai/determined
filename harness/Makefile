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
check: gen
	isort --check-only --skip client
	black . --check
	flake8
	mypy . --show-error-context --show-error-codes

.PHONY: test
test: gen
	pytest -v --runslow --durations=0 tests

.PHONY: test-tf2
test-tf2:
	pip freeze | grep "tensorflow==2.*"
	pytest -v --runslow --durations=0 tests/experiment/tensorflow
	# We must run these tests separately becuase estimators need to disable v2
	# behavior (a global operation). We are explicitly testing eager execution
	# for tf keras which needs v2 behavior enabled. You can't enable v2 behavior
	# anywhere but the "start" of your program. See:
	# https://github.com/tensorflow/tensorflow/issues/18304#issuecomment-379435515.
	pytest -v --runslow --durations=0 tests/experiment/keras

.PHONY: clean
clean:
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	rm -rf *.egg-info/
	rm -rf pip-wheel-metadata/
	rm -rf dist/
	rm -rf build/
	find . \( -name __pycache__ -o -name \*.pyc \) -delete

.PHONY: upload-try-now-template
upload-try-now-template: TRY_NOW_TEMPLATE = simple.yaml
upload-try-now-template: TRY_NOW_URL := s3://determined-ai-public/$(TRY_NOW_TEMPLATE)
upload-try-now-template: TEMPLATE_PATH := determined/deploy/aws/templates/$(TRY_NOW_TEMPLATE)
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
check-gen: force-gen $(GENERATED_COMMITTED)
	# Checking that committed, generated code is up-to-date by ensuring that
	# git reports the files as unchanged after forcibly regenerating the files:
	test -z "$(shell git status --porcelain $(GENERATED_COMMITTED))"

determined/common/schemas/expconf/_gen.py: $(GENERATION_INPUTS)
	../schemas/gen.py python\
		--package expconf \
		--output $@

.PHONY: gen-deploy-aws-vcpu-mapping
gen-deploy-aws-vcpu-mapping:
	python -m determined.deploy.aws.gen_vcpu_mapping determined/deploy/aws/vcpu_mapping.yaml
