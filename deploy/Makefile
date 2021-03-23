.PHONY: build
build:
	python setup.py -q bdist_wheel

.PHONY: publish
publish: build
	twine upload --verbose --non-interactive dist/*

.PHONY: clean
clean:
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	rm -rf *.egg-info/
	rm -rf pip-wheel-metadata/
	rm -rf dist/
	rm -rf build/
	find . \( -name __pycache__ -o -name \*.pyc \) -delete

.PHONY: fmt
fmt:
	isort -y
	black .

.PHONY: check
check:
	isort --check-only
	black . --check
	flake8
	mypy .
