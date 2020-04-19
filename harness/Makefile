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

BLACK_CONFIG_FILE := ../pyproject.toml
BLACK_COMMAND := black --config $(BLACK_CONFIG_FILE) .

fmt:
	$(BLACK_COMMAND)

check:
	$(BLACK_COMMAND) --check
	flake8 --config ../.flake8
	mypy . --config ../mypy.ini
