.PHONY: get-deps build clean graphql live check-fmt test fmt check

BUILDDIR ?= ../../build
WEBUI_ROOT := $(BUILDDIR)/share/determined/master/webui
CSS_OUT := $(WEBUI_ROOT)/elm/public/styles.css
ELM_OUT := $(WEBUI_ROOT)/elm/public/determined-ui.js

get-deps:
	for i in `seq 3`; do \
		yarn install --non-interactive --pure-lockfile --mutex file:/tmp/.yarn-mutex && break ; \
	done

# Elm and PostCSS will both create the directories to hold their output if necessary.

$(CSS_OUT): styles-in.css tailwind.config.js
	yarn run postcss styles-in.css -o $@

build: $(CSS_OUT)
	cp -r public $(WEBUI_ROOT)/elm
	yarn run elm make src/Main.elm --output="$(ELM_OUT)"

clean:
	rm -f yarn-error.log "$(CSS_OUT)" "$(ELM_OUT)"
	rm -rf elm-stuff/ tests/elm-stuff/ node_modules/

live: $(CSS_OUT)
	yarn run elm-live-workaround src/Main.elm --output="$(ELM_OUT)"

fmt:
	yarn run elm-format --yes src/ tests/

check-fmt:
	yarn run elm-format --validate src/ tests/

check: check-fmt
	yarn run elm-analyse

test:
	yarn run elm-test

graphql:
	rm -rf src/DetQL/
	yarn run elm-graphql --base DetQL --scalar-codecs CustomScalarCodecs --introspection-file ../../master/graphql-schema.json
