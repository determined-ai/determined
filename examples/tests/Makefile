.PHONY: fmt check

fmt:
	isort .
	black .

check:
	isort . --check-only
	black . --check
	flake8
