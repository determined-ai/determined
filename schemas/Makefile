all:

.PHONY: fmt
fmt:
	./lint.py `find expconf -name '*.json'`
	isort -y
	black .

.PHONY: check
check:
	./lint.py --check `find expconf -name '*.json'`
	isort --check-only
	black . --check
	flake8
	mypy `find -name '*.py'`
