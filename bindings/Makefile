SWAGGER_SPEC := ../proto/build/swagger/determined/api/v1/api.swagger.json
CODEGEN_BIN := swagger-codegen-cli-2.4.14.jar
ts_bindings_dest = ../webui/react/src/services/api-ts-sdk

.PHONY: all
all: get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: deps/$(CODEGEN_BIN)

deps/$(CODEGEN_BIN):
	mkdir -p deps
	curl https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.14/$(CODEGEN_BIN) \
		-o deps/$(CODEGEN_BIN)

build/typescript-fetch: $(SWAGGER_SPEC) deps/$(CODEGEN_BIN)
	mkdir -p build/typescript-fetch
	java -jar deps/$(CODEGEN_BIN) generate -i $(SWAGGER_SPEC) \
		-l typescript-fetch -o build/typescript-fetch
	touch build/typescript-fetch
	$(MAKE) copy-bindings

.PHONY: copy-bindings
copy-bindings:
	rm -rf $(ts_bindings_dest)/*
	cp -r build/typescript-fetch/* $(ts_bindings_dest)

.PHONY: build
build: build/typescript-fetch
	# This target has a dependency on ../proto/ that isn't captured here for CI's sake as they'd need
	# different dependencies in their environement.
	# $(MAKE) -C ../proto build

.PHONY: check
check: build
	$(MAKE) copy-bindings
	# Checking that the committed, generated code is up-to-date by ensuring that
	# Git reports the files as unchanged after forcibly regenerating them.
	# The API bindings generated here depend on changes to the API and after any changes to the API
	# you need to build the bindings and commit the newly generated code to avoid failing this check.
	test -z "$(shell git status --porcelain $(ts_bindings_dest)/*)"

.PHONY: clean-deps
clean-deps:
	rm -rf deps/

.PHONY: clean
clean:
	rm -rf build/

.PHONY: clean-all
clean-all: clean clean-deps
