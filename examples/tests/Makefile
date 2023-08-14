.PHONY: fmt check

fmt:
	isort .
	black .

check:
	isort . --check-only
	black . --check
	python -m flake8
