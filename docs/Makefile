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

../proto/build/swagger/determined/api/v1/api.swagger.json:
	$(MAKE) -C ../proto build

build/helm.stamp: ../helm/build/stamp
	rm -rf site/downloads/helm
	mkdir -p site/downloads/helm
	cp ../helm/build/*.tgz site/downloads/helm
	mkdir -p build
	touch $@

build/examples.stamp: ../examples/build/stamp
	rm -rf site/downloads/examples
	mkdir -p site/downloads/examples
	cp ../examples/build/*.tgz site/downloads/examples
	mkdir -p build
	touch $@

attributions.rst: $(shell find ../tools/scripts/licenses -type f)
	../tools/scripts/gen-attributions.py sphinx $@

build/sp-%.stamp: $(SPHINXWRAPPER) conf.py attributions.rst $(shell find . -name site -prune -o -type f \( -name '*.rst' -o -name '*.html' -o -name '*.css' \) -print) $(shell find assets -type f) $(shell find _templates -type f) build/examples.stamp build/helm.stamp build/swagger.stamp
	$(SPHINXWRAPPER) $(SPHINXBUILD) -M $* . site $(SPHINXOPTS)
	mkdir -p build
	touch $@

build/swagger.stamp: ../proto/build/swagger/determined/api/v1/api.swagger.json swagger-ui/*
	rm -rf site/html/rest-api
	mkdir -p site/html
	cp -r ./swagger-ui site/html/rest-api
	cp $< site/html/rest-api/
	mkdir -p build
	touch $@

.PHONY: build
build: build/sp-html.stamp

.PHONY: xml
xml: build/sp-xml.stamp

.PHONY: upload-search-index
upload-search-index: build/sp-xml.stamp
	python3 deploy/scrape.py --upload

.PHONY: clean
clean:
	rm -rf site build attributions.rst

.PHONY: pre-publish
pre-publish:
	cp ./swagger-ui/swagger-ui-main-deploy.js ./site/html/rest-api/swagger-ui-main.js

.PHONY: publish-check
	python3 redirects.py publish-check

.PHONY: publish
publish: pre-publish publish-check
	$(MAKE) -C deploy publish

.PHONY: preview
preview: build
	python deploy/upload.py --preview

live: build/sp-html.stamp
	python3 live.py

RSTFMT_WIDTH = 100

.PHONY: fmt
fmt:
	git ls-files -z '*.rst' | xargs -0 rstfmt -w $(RSTFMT_WIDTH)

.PHONY: check
check: attributions.rst
	git ls-files -z '*.rst' | xargs -0 rstfmt -w $(RSTFMT_WIDTH) --check
	python3 redirects.py check

.PHONY: get-deps
get-deps:
	pip install -r requirements.txt
