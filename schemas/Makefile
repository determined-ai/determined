all:

.PHONY: fmt
fmt:
	./lint.py expconf/v1/*
	isort -y
	black .

.PHONY: check
check:
	./lint.py --check expconf/v1/*
	isort --check-only
	black . --check
	flake8
	mypy `find . -name '*.py'`
