GENERATED_COMMITTED := determined_common/schemas/expconf/_gen.py
GENERATION_INPUTS = ../schemas/gen.py $(shell find ../schemas/expconf -name '*.json')

.PHONY: clean
clean:
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/
	rm -rf *.egg-info/
	rm -rf pip-wheel-metadata/
	rm -rf dist/
	rm -rf build/
	find . \( -name __pycache__ -o -name \*.pyc \) -delete

.PHONY: ungen
ungen:
	git checkout -- $(GENERATED_COMMITTED)

.PHONY: gen
gen: $(GENERATED_COMMITTED)

.PHONY: force-gen
force-gen:
	touch ../schemas/gen.py

.PHONY: check-gen
check-gen: force-gen $(GENERATED_COMMITTED)
	# Checking that committed, generated code is up-to-date by ensuring that
	# git reports the files as unchanged after forcibly regenerating the files:
	test -z "$(shell git status --porcelain $(GENERATED_COMMITTED))"

determined_common/schemas/expconf/_gen.py: $(GENERATION_INPUTS)
	../schemas/gen.py python\
		--package expconf \
		--output $@

.PHONY: build
build:
	python setup.py -q bdist_wheel

.PHONY: publish
publish:
	twine upload --verbose --non-interactive dist/*

.PHONY: fmt
fmt:
	isort -y
	black . --exclude '_gen.py'

.PHONY: check
check: gen
	isort --check-only
	black . --check --exclude '_gen.py'
	flake8 --exclude '_gen.py'
	mypy .

.PHONY: test
test: gen
	pytest -v tests
