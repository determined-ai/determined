EXAMPLES := $(wildcard official/*/.)
EXAMPLES_DIRS := $(patsubst official/%/., build/%.tgz, $(EXAMPLES))

.PHONY: clean
clean:
	find . \( -name __pycache__ -o -name \*.pyc -o -name .mypy_cache \) -print0 | xargs -0 rm -rf
	rm -rf build/

build/%.tgz: official/%/
	find "$<" \( -name __pycache__ -o -name \*.pyc \) -delete
	mkdir -p $$(dirname "$@")
	tar -czf "$@" -C $$(dirname "$<") $$(basename "$<")

test:
	pytest -vv -s --durations=0 tests

.PHONY: build
build: $(EXAMPLES_DIRS)
