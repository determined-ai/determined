.PHONY: clean build publish fmt check test

VERSION := $(shell cat ../VERSION)
BUILDDIR ?= ../build
DIST_DIRECTORY := "$(BUILDDIR)"/share/determined/master/wheels
HARNESS_WHEEL := $(DIST_DIRECTORY)/determined-$(VERSION)-py3-none-any.whl

clean:
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	rm -rf *.egg-info/
	rm -rf pip-wheel-metadata/
	rm -rf $(DIST_DIRECTORY)
	rm -rf build/
	find . \( -name __pycache__ -o -name \*.pyc \) -delete

build:
	python setup.py -q bdist_wheel -d $(DIST_DIRECTORY)

publish:
	twine upload --non-interactive $(HARNESS_WHEEL)

fmt:
	isort -y
	black .

check:
	isort --check-only
	black . --check
	flake8
	mypy .

test:
	pytest -v --runslow --durations=0 tests

test-tf2:
	pip freeze | grep "tensorflow==2.*"
	pytest -v --runslow --durations=0 tests/experiment/tensorflow
	# We must run these tests separately becuase estimators need to disable v2
	# behavior (a global operation). We are explicitly testing eager execution
	# for tf keras which needs v2 behavior enabled. You can't enable v2 behavior
	# anywhere but the "start" of your program. See:
	# https://github.com/tensorflow/tensorflow/issues/18304#issuecomment-379435515.
	pytest -v --runslow --durations=0 tests/experiment/keras
