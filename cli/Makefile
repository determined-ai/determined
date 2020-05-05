.PHONY: clean build publish fmt check test

VERSION := $(shell cat ../VERSION)
BUILDDIR ?= ../build
DIST_DIRECTORY := "$(BUILDDIR)"/share/determined/master/wheels
CLI_WHEEL := $(DIST_DIRECTORY)/determined_cli-$(VERSION)-py3-none-any.whl

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
	twine upload --non-interactive $(CLI_WHEEL)

fmt:
	isort -y
	black .

check:
	isort --check-only
	black . --check
	flake8
	mypy .

test:
	pytest tests
