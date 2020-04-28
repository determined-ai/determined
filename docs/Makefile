.PHONY: build clean help sphinx publish ALWAYS

export VERSION := $(shell cat ../VERSION)

BUILDDIR ?= ../build

EXAMPLES := $(wildcard ../examples/official/*/.)
EXAMPLES_DIRS := $(patsubst ../examples/official/%/., $(BUILDDIR)/docs-downloads/examples/%.tgz, $(EXAMPLES))

# You can set these variables from the command line.
SPHINXOPTS    = -W
SPHINXBUILD   = sphinx-build
SPHINXPROJ    = Determined
SOURCEDIR     = .
# The intermediate place where we tell Sphinx to put the full output.
OUTDIR        = "$(BUILDDIR)"/docs-output
# The final place where we put just the HTML output that the master wants.
TARGETDIR     = "$(BUILDDIR)"/share/determined/master/webui/docs

# When available, we use some GNU tar options that help reproducibility and
# Docker caching.
ifeq (0,$(shell tar --sort=name --mtime='2018-01-01 00:00:00' -cf /dev/null -T /dev/null > /dev/null 2>&1; echo $$?))
	TAR := GZIP=-n tar --mtime='2018-01-01 00:00:00' --sort=name
else
	TAR := tar
endif

build: $(EXAMPLES_DIRS) sp-html
	rm -rf $(TARGETDIR)
	mkdir -p $$(dirname $(TARGETDIR))
	cp -r $(OUTDIR)/html $(TARGETDIR)

$(BUILDDIR)/docs-downloads/examples/%.tgz: ../examples/official/%/
	find "$<" \( -name __pycache__ -o -name \*.pyc \) -delete
	mkdir -p $$(dirname "$@")
	$(TAR) -czf "$@" -C $$(dirname "$<") $$(basename "$<")

clean:
	rm -rf $(TARGETDIR) $(OUTDIR)

publish: clean
	$(MAKE) build
	$(MAKE) -C deploy publish

# Catch-all target: route all unknown targets to Sphinx using the new
# "make mode" option.  $(O) is meant as a shortcut for $(SPHINXOPTS).
sp-%: ALWAYS
	@$(SPHINXBUILD) -M $* "$(SOURCEDIR)" "$(OUTDIR)" $(SPHINXOPTS) $(O)
