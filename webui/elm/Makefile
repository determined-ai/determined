.PHONY: get-deps build clean graphql live check-fmt test fmt check

BUILDDIR ?= ../../build
WEBUI_ROOT := $(BUILDDIR)/share/determined/master/webui
CSS_OUT := $(WEBUI_ROOT)/elm/public/styles.css
ELM_OUT := $(WEBUI_ROOT)/elm/public/determined-ui.js

get-deps:
	for i in `seq 3`; do \
		npm ci --no-audit && break ; \
	done

# Elm and PostCSS will both create the directories to hold their output if necessary.

$(CSS_OUT): styles-in.css tailwind.config.js
	npm run build-css -- -o $@

build: $(CSS_OUT)
	mkdir -p $(WEBUI_ROOT)/elm
	cp -r public $(WEBUI_ROOT)/elm
	npm run build-elm -- --output="$(ELM_OUT)"

clean:
	rm -f npm-debug.log "$(CSS_OUT)" "$(ELM_OUT)"
	rm -rf elm-stuff/ tests/elm-stuff/ node_modules/

live: $(CSS_OUT)
	npm run live-workaround -- --output="$(ELM_OUT)"

fmt:
	npm run format -- --yes src/ tests/

check-fmt:
	npm run format -- --validate src/ tests/

check: check-fmt
	npm run analyse

test:
	npm run test

graphql:
	rm -rf src/DetQL/
	npm run graphql
