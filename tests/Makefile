.PHONY: fmt check

fmt:
	isort -y
	black .

check:
	isort --check-only
	black . --check
	flake8
	mypy cli
	mypy unit
	mypy integrations
