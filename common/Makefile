.PHONY: clean build publish fmt check test

# TODO: It may make sense to ultimately have this (and the CLI and harness) not
# build directly into the master directory, supposing we'll want to have them
# for purposes besides shipping them out from the master. But this simplifies
# things a bit for now.

VERSION := $(shell cat ../VERSION)
BUILDDIR ?= ../build
DIST_DIRECTORY := "$(BUILDDIR)"/share/determined/master/wheels
COMMON_WHEEL := $(DIST_DIRECTORY)/determined_common-$(VERSION)-py3-none-any.whl

clean:
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	rm -rf *.egg-info/
	rm -rf build/
	find . \( -name __pycache__ -o -name \*.pyc \) -delete

build:
	python setup.py -q bdist_wheel -d $(DIST_DIRECTORY)

publish:
	twine upload --non-interactive $(COMMON_WHEEL)

fmt:
	isort -y
	black .

check:
	isort --check-only
	black . --check
	flake8
	mypy .
