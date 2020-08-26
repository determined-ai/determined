export VERSION := $(shell cat ../VERSION)

SPHINXOPTS    = -W
SPHINXBUILD   = sphinx-build

.PHONY: build
build:
	mkdir -p site/downloads/examples
	cp ../examples/build/* site/downloads/examples
	mkdir -p site/downloads/helm
	cp ../helm/build/* site/downloads/helm
	$(MAKE) sp-html

.PHONY: clean
clean:
	rm -rf site

.PHONY: publish
publish: clean
	$(MAKE) build
	$(MAKE) -C deploy publish

live:
	npx nodemon --ext txt --exec "$(MAKE) build" --ignore site

# Catch-all target: route all unknown targets to Sphinx using the new
# "make mode" option.  $(O) is meant as a shortcut for $(SPHINXOPTS).
.PHONY: ALWAYS
sp-%: ALWAYS
	@$(SPHINXBUILD) -M $* . site $(SPHINXOPTS) $(O)
