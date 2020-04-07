.PHONY: clean build publish fmt check test

VERSION := $(shell cat ../VERSION)
BUILDDIR ?= ../build
DEPLOY_WHEEL := $(BUILDDIR)/determined_deploy-$(VERSION)-py3-none-any.whl
DEPLOY_WHEEL_URL := s3://determined-ai-artifacts/deploy/$(shell basename $(DEPLOY_WHEEL))
DEPLOY_WHEEL_LATEST_URL := s3://determined-ai-artifacts/deploy/determined_deploy-latest-py3-none-any.whl

build:
	python setup.py bdist_wheel -d $(DIST_DIRECTORY)

publish:
	aws s3 ls $(DEPLOY_WHEEL_URL); \
	if [ $$? -eq 0 ] ; then \
	  echo "Cannot overwrite existing package: $(DEPLOY_WHEEL_URL)"; exit 1; \
	fi

	aws s3 cp --content-type "binary/octet-stream" $(DEPLOY_WHEEL) $(DEPLOY_WHEEL_URL)
	aws s3 cp --content-type "binary/octet-stream" $(DEPLOY_WHEEL) $(DEPLOY_WHEEL_LATEST_URL)

clean:
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	rm -rf *.egg-info/
	rm -rf pip-wheel-metadata/
	rm -rf dist/
	rm -rf build/
	find . \( -name __pycache__ -o -name \*.pyc \) -delete

build:
	python setup.py bdist_wheel -d "$(BUILDDIR)"

BLACK_CONFIG_FILE := ../pyproject.toml
BLACK_COMMAND := black --config $(BLACK_CONFIG_FILE) .

fmt:
	$(BLACK_COMMAND)

check:
	$(BLACK_COMMAND) --check
	flake8 --config ../.flake8
	mypy . --config ../mypy.ini
