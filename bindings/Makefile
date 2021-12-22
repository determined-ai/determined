SWAGGER_SPEC := ../proto/build/swagger/determined/api/v1/api.swagger.json
CODEGEN_BIN := swagger-codegen-cli-2.4.14.jar
py_bindings_dest := ../harness/determined/common/api/bindings.py
ts_bindings_dest := ../webui/react/src/services/api-ts-sdk

.PHONY: all
all: get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: deps/$(CODEGEN_BIN)

deps/$(CODEGEN_BIN):
	mkdir -p deps
	curl https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.14/$(CODEGEN_BIN) \
		-o deps/$(CODEGEN_BIN)

.PHONY: bindings-copy-over
bindings-copy-over:
	rm -rf $(ts_bindings_dest)
	cp -r  build/typescript-fetch $(ts_bindings_dest)

.PHONY: build/python
build/python: $(SWAGGER_SPEC)
	python generate_bindings.py > $(py_bindings_dest)

build/%: $(SWAGGER_SPEC) deps/$(CODEGEN_BIN)
	mkdir -p build/$*
	java -jar deps/$(CODEGEN_BIN) generate -i $(SWAGGER_SPEC) \
		-l typescript-fetch -o build/typescript-fetch
	touch build/typescript-fetch
	$(MAKE) bindings-copy-over

.PHONY: build
build: build/typescript-fetch build/python

.PHONY: check
check: check/python check/typescript-fetch
	# Checking that the (committed) generated code is up-to-date by ensuring that
	# git reports the files as unchanged after forcibly regenerating them.
	# WARNING this has a dependency on proto module being built recently
	# which isn't linked here.

.PHONY: check-python
check/python: build/python
	test -z "$(shell git status --porcelain $(py_bindings_dest))"

.PHONY: check-typescript-fetch
check/typescript-fetch: build/typescript-fetch
	test -z "$(shell git status --porcelain $(ts_bindings_dest)/*)"

.PHONY: clean-deps
clean-deps:
	rm -rf deps/

.PHONY: clean
clean:
	rm -rf build/

.PHONY: clean-all
clean-all: clean clean-deps
