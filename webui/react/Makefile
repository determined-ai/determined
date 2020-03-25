.PHONY: get-deps build clean check test fmt test check-fmt live 

BUILDDIR ?= ../../build
WEBUI_ROOT := $(BUILDDIR)/share/determined/master/webui

get-deps:
	for i in `seq 3`; do \
		npm ci --no-audit && break ; \
	done

# HACK: It is practically impossible to get the React thing to build into
# anywhere besides `build`. Trick it by symlinking the desired location to
# `build`.
build:
	rm -rf build
	mkdir -p "$(WEBUI_ROOT)"/react
	ln -s "$(WEBUI_ROOT)"/react build
	npm run build

clean:
	rm -rf build node_modules/

live:
	npm start

check:
	npm run lint

check-fmt: ;

fmt:
	npm run lint:js -- --fix
	npm run lint:css -- --fix

test:
	npm run test -- --watchAll=false
