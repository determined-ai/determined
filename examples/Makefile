TUTORIAL_EXAMPLES := $(wildcard tutorials/*/.)
TUTORIAL_EXAMPLES_DIRS := $(patsubst tutorials/%/., build/%.tgz, $(TUTORIAL_EXAMPLES))

CV_EXAMPLES := $(wildcard computer-vision/*/.)
CV_EXAMPLES_DIRS := $(patsubst computer-vision/%/., build/%.tgz, $(CV_EXAMPLES))

NLP_EXAMPLES := $(wildcard NLP/*/.)
NLP_EXAMPLES_DIRS := $(patsubst NLP/%/., build/%.tgz, $(NLP_EXAMPLES))

HPSEARCH_EXAMPLES := $(wildcard hp-search-benchmarks/*/.)
HPSEARCH_EXAMPLES_DIRS := $(patsubst hp-search-benchmarks/%/., build/%.tgz, $(HPSEARCH_EXAMPLES))

NAS_EXAMPLES := $(wildcard NAS/*/.)
NAS_EXAMPLES_DIRS := $(patsubst NAS/%/., build/%.tgz, $(NAS_EXAMPLES))

META_LEARNING_EXAMPLES := $(wildcard meta-learning/*/.)
META_LEARNING_EXAMPLES_DIRS := $(patsubst meta-learning/%/., build/%.tgz, $(META_LEARNING_EXAMPLES))

GAN_EXAMPLES := $(wildcard GAN/*/.)
GAN_EXAMPLES_DIRS := $(patsubst GAN/%/., build/%.tgz, $(GAN_EXAMPLES))

.PHONY: build
build: $(TUTORIAL_EXAMPLES_DIRS) $(CV_EXAMPLES_DIRS) $(NLP_EXAMPLES_DIRS) $(HPSEARCH_EXAMPLES_DIRS) $(NAS_EXAMPLES_DIRS) $(META_LEARNING_EXAMPLES_DIRS) $(GAN_EXAMPLES_DIRS)

.PHONY: clean
clean:
	find . \( -name __pycache__ -o -name \*.pyc -o -name .mypy_cache \) -print0 | xargs -0 rm -rf
	rm -rf build/

build/%.tgz: */%/
	find "$<" \( -name __pycache__ -o -name \*.pyc \) -delete
	mkdir -p $$(dirname "$@")
	tar -czf "$@" -C $$(dirname "$<") $$(basename "$<")

test:
	pytest -vv -s --durations=0 tests

fmt:
	make -C tests fmt

check:
	make -C tests check
