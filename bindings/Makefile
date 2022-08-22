SWAGGER_SPEC := ../proto/build/swagger/determined/api/v1/api.swagger.json
CODEGEN_VER = 2.4.27
CODEGEN_BIN := swagger-codegen-cli-$(CODEGEN_VER).jar
py_bindings_dest := ../harness/determined/common/api/bindings.py
ts_bindings_dest := ../webui/react/src/services/api-ts-sdk
py_generator := generate_bindings.py

.PHONY: all
all: get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: deps/$(CODEGEN_BIN)

deps/$(CODEGEN_BIN):
	mkdir -p deps
	curl https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/$(CODEGEN_VER)/$(CODEGEN_BIN) \
		-o deps/$(CODEGEN_BIN)

.PHONY: force-gen
force-gen:
	touch $(SWAGGER_SPEC)

.PHONY: bindings-copy-over
bindings-copy-over:
	rm -rf $(ts_bindings_dest)
	cp -r  build/typescript-fetch $(ts_bindings_dest)

$(py_bindings_dest): $(SWAGGER_SPEC) $(py_generator)
	python $(py_generator) --output $(py_bindings_dest)

.PHONY: build/python
build/python: $(py_bindings_dest)

build/%: $(SWAGGER_SPEC) deps/$(CODEGEN_BIN)
	mkdir -p build/$*
	java -jar deps/$(CODEGEN_BIN) generate -i $(SWAGGER_SPEC) \
		-l typescript-fetch -o build/typescript-fetch
	$(MAKE) bindings-copy-over
	touch build/typescript-fetch

.PHONY: build
build: build/typescript-fetch $(py_bindings_dest)

.PHONY: check
check: force-gen check/python check/typescript-fetch
	# Checking that the (committed) generated code is up-to-date by ensuring that
	# git reports the files as unchanged after forcibly regenerating them.
	# WARNING this has a dependency on proto module being built recently
	# which isn't linked here.

.PHONY: check-python
check/python: build/python
	test -z "$(shell git status --porcelain $(py_bindings_dest))" || (git diff; false)

.PHONY: check-typescript-fetch
check/typescript-fetch: build/typescript-fetch
	test -z "$(shell git status --porcelain $(ts_bindings_dest)/*)" || (git diff; false)

.PHONY: clean-deps
clean-deps:
	rm -rf deps/

.PHONY: clean
clean:
	rm -rf build/ $(py_bindings_dest) $(ts_bindings_dest)

.PHONY: clean-all
clean-all: clean clean-deps
