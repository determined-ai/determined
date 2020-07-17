NATIVE_EXAMPLES := $(wildcard official/native/*/.)
NATIVE_EXAMPLES_DIRS := $(patsubst official/native/%/., build/%.tgz, $(NATIVE_EXAMPLES))

TRIAL_EXAMPLES := $(wildcard official/trial/*/.)
TRIAL_EXAMPLES_DIRS := $(patsubst official/trial/%/., build/%.tgz, $(TRIAL_EXAMPLES))

.PHONY: build
build: $(NATIVE_EXAMPLES_DIRS) $(TRIAL_EXAMPLES_DIRS)

.PHONY: clean
clean:
	find . \( -name __pycache__ -o -name \*.pyc -o -name .mypy_cache \) -print0 | xargs -0 rm -rf
	rm -rf build/

build/%.tgz: official/*/%/
	find "$<" \( -name __pycache__ -o -name \*.pyc \) -delete
	mkdir -p $$(dirname "$@")
	tar -czf "$@" -C $$(dirname "$<") $$(basename "$<")

test:
	pytest -vv -s --durations=0 tests

fmt:
	make -C tests fmt

check:
	make -C tests check
