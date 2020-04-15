.PHONY: clean build publish fmt check test

VERSION := $(shell cat ../VERSION)
BUILDDIR ?= ../build
DIST_DIRECTORY := "$(BUILDDIR)"/share/determined/master/wheels
HARNESS_WHEEL := $(DIST_DIRECTORY)/determined-$(VERSION)-py3-none-any.whl
HARNESS_WHEEL_URL := s3://determined-ai-artifacts/determined/$(shell basename $(HARNESS_WHEEL))
HARNESS_WHEEL_LATEST_URL := s3://determined-ai-artifacts/determined/determined-latest-py3-none-any.whl

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
	aws s3 ls $(HARNESS_WHEEL_URL); \
	if [ $$? -eq 0 ] ; then \
	  echo "Cannot overwrite existing package: $(HARNESS_WHEEL_URL)"; exit 1; \
	fi

	aws s3 cp --content-type "binary/octet-stream" $(HARNESS_WHEEL) $(HARNESS_WHEEL_URL)
	aws s3 cp --content-type "binary/octet-stream" $(HARNESS_WHEEL) $(HARNESS_WHEEL_LATEST_URL)
	twine upload --non-interactive $(HARNESS_WHEEL)

BLACK_CONFIG_FILE := ../pyproject.toml
BLACK_COMMAND := black --config $(BLACK_CONFIG_FILE) .

fmt:
	$(BLACK_COMMAND)

check:
	$(BLACK_COMMAND) --check
	flake8 --config ../.flake8
	mypy . --config ../mypy.ini
