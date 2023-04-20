export VERSION := $(shell cat ../VERSION)
.DEFAULT_GOAL := build

SPHINXOPTS    = -W
SPHINXBUILD   = sphinx-build
SPHINXWRAPPER = ./sphinx-wrapper.py

# These are "shallow" dependencies, where we check the output of another module
# but not the dependencies of that output.  This will catch the "nothing has
# been built" and the "module was rebuilt" cases but not the "module needs
# rebuilding" case.
../helm/build/stamp:
	$(MAKE) -C ../helm build/stamp

../examples/build/stamp:
	$(MAKE) -C ../examples build/stamp

../model_hub/build-examples/stamp:
	$(MAKE) -C ../model_hub build-examples/stamp

../proto/build/swagger/determined/api/v1/api.swagger.json:
	$(MAKE) -C ../proto build

build/helm.stamp: ../helm/build/stamp
	rm -rf site/downloads/helm
	mkdir -p site/downloads/helm
	cp ../helm/build/*.tgz site/downloads/helm
	mkdir -p build
	touch $@

build/examples.stamp: ../examples/build/stamp ../model_hub/build-examples/stamp
	rm -rf site/downloads/examples
	mkdir -p site/downloads/examples
	cp ../examples/build/*.tgz site/downloads/examples
	cp ../model_hub/build-examples/*.tgz site/downloads/examples
	mkdir -p build
	touch $@

attributions.rst: $(shell find ../tools/scripts/licenses -type f)
	../tools/scripts/gen-attributions.py sphinx $@

build/sp-%.stamp: $(SPHINXWRAPPER) conf.py attributions.rst $(shell find . -name site -prune -o -type f \( -name '*.rst' -o -name '*.html' -o -name '*.css' \) -print) $(shell find assets -type f) $(shell find _templates -type f)
	@$(SPHINXWRAPPER) $(SPHINXBUILD) -M $* . site $(SPHINXOPTS)
	mkdir -p build
	touch $@

build/swagger.stamp: ../proto/build/swagger/determined/api/v1/api.swagger.json swagger-ui/*
	rm -rf site/html/rest-api
	mkdir -p site/html
	cp -r ./swagger-ui site/html/rest-api
	cp $< site/html/rest-api/
	mkdir -p build
	touch $@

build/stamp: build/examples.stamp build/helm.stamp build/swagger.stamp build/sp-html.stamp
	touch $@

.PHONY: build
build: build/stamp

.PHONY: xml
xml: build/sp-xml.stamp

.PHONY: clean
clean:
	rm -rf site build attributions.rst

.PHONY: pre-publish
pre-publish:
	cp ./swagger-ui/swagger-ui-main-deploy.js ./site/html/rest-api/swagger-ui-main.js

.PHONY: publish
publish: pre-publish
	$(MAKE) -C deploy publish

live: build/stamp
	python3 live.py

RSTFMT_WIDTH = 100

.PHONY: fmt
fmt:
	git ls-files -z '*.rst' | xargs -0 rstfmt -w $(RSTFMT_WIDTH)

.PHONY: check
check:
	git ls-files -z '*.rst' | xargs -0 rstfmt -w $(RSTFMT_WIDTH) --check
