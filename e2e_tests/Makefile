.PHONY: fmt check test

fmt:
	isort
	black .

check:
	isort --check-only
	black . --check
	flake8
	mypy tests

test:
	pytest -vv -s --durations=0 tests
