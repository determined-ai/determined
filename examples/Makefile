TUTORIAL_EXAMPLES := $(wildcard tutorials/*/.)
TUTORIAL_EXAMPLES_DIRS := $(patsubst tutorials/%/., build/%.tgz, $(TUTORIAL_EXAMPLES))

CV_EXAMPLES := $(wildcard computer_vision/*/.)
CV_EXAMPLES_DIRS := $(patsubst computer_vision/%/., build/%.tgz, $(CV_EXAMPLES))

NLP_EXAMPLES := $(wildcard nlp/*/.)
NLP_EXAMPLES_DIRS := $(patsubst nlp/%/., build/%.tgz, $(NLP_EXAMPLES))

HPSEARCH_EXAMPLES := $(wildcard hp_search_benchmarks/*/.)
HPSEARCH_EXAMPLES_DIRS := $(patsubst hp_search_benchmarks/%/., build/%.tgz, $(HPSEARCH_EXAMPLES))

NAS_EXAMPLES := $(wildcard nas/*/.)
NAS_EXAMPLES_DIRS := $(patsubst nas/%/., build/%.tgz, $(NAS_EXAMPLES))

META_LEARNING_EXAMPLES := $(wildcard meta_learning/*/.)
META_LEARNING_EXAMPLES_DIRS := $(patsubst meta_learning/%/., build/%.tgz, $(META_LEARNING_EXAMPLES))

GAN_EXAMPLES := $(wildcard gan/*/.)
GAN_EXAMPLES_DIRS := $(patsubst gan/%/., build/%.tgz, $(GAN_EXAMPLES))

DT_EXAMPLES := $(wildcard decision_trees/*/.)
DT_EXAMPLES_DIRS := $(patsubst decision_trees/%/., build/%.tgz, $(DT_EXAMPLES))

FEATURES_EXAMPLES := $(wildcard features/*/.)
FEATURES_EXAMPLES_DIRS := $(patsubst features/%/., build/%.tgz, $(FEATURES_EXAMPLES))

.PHONY: build
build: $(TUTORIAL_EXAMPLES_DIRS) $(CV_EXAMPLES_DIRS) $(NLP_EXAMPLES_DIRS) $(HPSEARCH_EXAMPLES_DIRS) $(NAS_EXAMPLES_DIRS) $(META_LEARNING_EXAMPLES_DIRS) $(GAN_EXAMPLES_DIRS) $(DT_EXAMPLES_DIRS) $(FEATURES_EXAMPLES_DIRS)

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
	black .
# 	isort -y

check:
	make -C tests check
	black . --check
# 	isort --check-only
	mypy `find . -name '*.py'`
