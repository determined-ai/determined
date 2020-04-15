.PHONY: clean build publish fmt check test

VERSION := $(shell cat ../VERSION)
BUILDDIR ?= ../build
DIST_DIRECTORY := "$(BUILDDIR)"/share/determined/master/wheels
CLI_WHEEL := $(DIST_DIRECTORY)/determined_cli-$(VERSION)-py3-none-any.whl
CLI_WHEEL_URL := s3://determined-ai-artifacts/cli/$(shell basename $(CLI_WHEEL))
CLI_WHEEL_LATEST_URL := s3://determined-ai-artifacts/cli/determined_cli-latest-py3-none-any.whl

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
	aws s3 ls $(CLI_WHEEL_URL); \
	if [ $$? -eq 0 ] ; then \
	  echo "Cannot overwrite existing package: $(CLI_WHEEL_URL)"; exit 1; \
	fi

	aws s3 cp --content-type "binary/octet-stream" $(CLI_WHEEL) $(CLI_WHEEL_URL)
	aws s3 cp --content-type "binary/octet-stream" $(CLI_WHEEL) $(CLI_WHEEL_LATEST_URL)
	twine upload --non-interactive $(CLI_WHEEL)

BLACK_CONFIG_FILE := ../pyproject.toml
BLACK_COMMAND := black --config $(BLACK_CONFIG_FILE) .

fmt:
	$(BLACK_COMMAND)

check:
	$(BLACK_COMMAND) --check
	flake8 --config ../.flake8
	mypy . --config ../mypy.ini

test:
	# TODO: Replace with tox.
	docker build -t determined-cli-deps -f ../tests/cli/Dockerfile.cli-deps ..
	docker run determined-cli-deps
	docker build -t determined-cli-tests -f ../tests/cli/Dockerfile.cli-tests ..
	docker run -t determined-cli-tests
