.PHONY: clean build publish upload-try-now-template fmt check test

VERSION := $(shell cat ../VERSION)
BUILDDIR ?= ../build
DEPLOY_WHEEL := $(BUILDDIR)/determined_deploy-$(VERSION)-py3-none-any.whl

build:
	python setup.py bdist_wheel -d $(DIST_DIRECTORY)

upload-try-now-template: TRY_NOW_TEMPLATE = simple.yaml
upload-try-now-template: TRY_NOW_URL := s3://determined-ai-public/$(TRY_NOW_TEMPLATE)
upload-try-now-template: TEMPLATE_PATH := determined_deploy/aws/templates/$(TRY_NOW_TEMPLATE)
upload-try-now-template:
	aws s3 cp $(TEMPLATE_PATH) $(TRY_NOW_URL) --acl public-read

publish: upload-try-now-template
	twine upload --non-interactive $(DEPLOY_WHEEL)

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
