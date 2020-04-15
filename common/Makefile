.PHONY: clean build publish fmt check test

# TODO: It may make sense to ultimately have this (and the CLI and harness) not
# build directly into the master directory, supposing we'll want to have them
# for purposes besides shipping them out from the master. But this simplifies
# things a bit for now.

VERSION := $(shell cat ../VERSION)
BUILDDIR ?= ../build
DIST_DIRECTORY := "$(BUILDDIR)"/share/determined/master/wheels
COMMON_WHEEL := $(DIST_DIRECTORY)/determined_common-$(VERSION)-py3-none-any.whl
COMMON_WHEEL_URL := s3://determined-ai-artifacts/common/$(shell basename $(COMMON_WHEEL))
COMMON_WHEEL_LATEST_URL := s3://determined-ai-artifacts/common/determined_common-latest-py3-none-any.whl

clean:
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	rm -rf *.egg-info/
	rm -rf build/
	find . \( -name __pycache__ -o -name \*.pyc \) -delete

build:
	python setup.py -q bdist_wheel -d $(DIST_DIRECTORY)

publish:
	aws s3 ls $(COMMON_WHEEL_URL); \
	if [ $$? -eq 0 ] ; then \
	  echo "Cannot overwrite existing package: $(COMMON_WHEEL_URL)"; exit 1; \
	fi

	aws s3 cp --content-type "binary/octet-stream" $(COMMON_WHEEL) $(COMMON_WHEEL_URL)
	aws s3 cp --content-type "binary/octet-stream" $(COMMON_WHEEL) $(COMMON_WHEEL_LATEST_URL)
	twine upload --non-interactive $(COMMON_WHEEL)

BLACK_CONFIG_FILE := ../pyproject.toml
BLACK_COMMAND := black --config $(BLACK_CONFIG_FILE) .

fmt:
	$(BLACK_COMMAND)

check:
	$(BLACK_COMMAND) --check
	flake8 --config ../.flake8
	mypy . --config ../mypy.ini
