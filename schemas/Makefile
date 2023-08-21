all:

.PHONY: fmt
fmt:
	./lint.py `find expconf -name '*.json'`
	isort .
	black .

.PHONY: check
check:
	./lint.py --check `find expconf -name '*.json'`
	isort . --check-only
	black . --check
	python -m flake8
	mypy `find . -name '*.py'`
