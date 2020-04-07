.PHONY: get-deps build clean check test fmt test check-fmt live 

BUILDDIR ?= ../../build
WEBUI_ROOT := $(BUILDDIR)/share/determined/master/webui

get-deps:
	for i in `seq 3`; do \
		yarn install --non-interactive --pure-lockfile --mutex file:/tmp/.yarn-mutex && break ; \
	done

# HACK: It is practically impossible to get the React thing to build into
# anywhere besides `build`. Trick it by symlinking the desired location to
# `build`.
build:
	rm -rf build
	mkdir -p "$(WEBUI_ROOT)"/react
	ln -s "$(WEBUI_ROOT)"/react build
	yarn run build

clean:
	rm -rf build node_modules/

live:
	yarn run start

check:
	yarn run lint

check-fmt: ;

fmt:
	yarn run lint:js --fix
	yarn run lint:css --fix

test:
	yarn run test --watchAll=false
