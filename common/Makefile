GENERATED := determined_common/schemas/expconf/_gen.py

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
	rm -f $(GENERATED)

gen: $(GENERATED)

determined_common/schemas/expconf/_gen.py: ../schemas/gen.py $(shell find ../schemas/expconf -name "*.json")
	../schemas/gen.py \
		--output $@ \
		python \
		$(shell find ../schemas/expconf -name "*.json")

.PHONY: build
build:
	python setup.py -q bdist_wheel

.PHONY: publish
publish:
	twine upload --verbose --non-interactive dist/*

.PHONY: fmt
fmt:
	isort -y
	black .

.PHONY: check
check: gen
	isort --check-only
	black . --check
	flake8 --exclude '*_gen.py'
	mypy .

.PHONY: test
test: gen
	pytest -v tests
