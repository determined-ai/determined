SWAGGER_SPEC := ../proto/build/swagger/determined/api/v1/api.swagger.json
py_bindings_dest := ../harness/determined/common/api/bindings.py
ts_bindings_dest := ../webui/react/src/services/api-ts-sdk
e2e_bindings_dest := ../webui/react/src/generated/orval/e2e-client.ts
py_generator := generate_bindings_py.py
ts_generator := generate_bindings_ts.py

.PHONY: all
all: build

.PHONY: force-gen
force-gen:
	touch $(SWAGGER_SPEC)

$(py_bindings_dest): $(SWAGGER_SPEC) swagger_parser.py $(py_generator)
	python $(py_generator) --output $(py_bindings_dest)

$(ts_bindings_dest): $(SWAGGER_SPEC) swagger_parser.py $(ts_generator) $(wildcard static_ts_files/*)
	python $(ts_generator) --output $@

$(e2e_bindings_dest): $(SWAGGER_SPEC)
	cd ../webui/react && npm run bindings

.PHONY: build/python
build/python: $(py_bindings_dest)

.PHONY: build/typescript
build/typescript: $(ts_bindings_dest)

.PHONY: build/e2e
build/e2e: $(ts_bindings_dest)

.PHONY: build
build: $(ts_bindings_dest) $(py_bindings_dest) $(e2e_bindings_dest)

.PHONY: check
check: force-gen check/python check/typescript check/e2e
	# Checking that the (committed) generated code is up-to-date by ensuring that
	# git reports the files as unchanged after forcibly regenerating them.
	# WARNING this has a dependency on proto module being built recently
	# which isn't linked here.

.PHONY: check/python
check/python: build/python
	test -z "$(shell git status --porcelain $(py_bindings_dest))" || (git diff; false)

.PHONY: check/typescript
check/typescript: build/typescript
	test -z "$(shell git status --porcelain $(ts_bindings_dest)/*)" || (git diff; false)

.PHONY: check/e2e
check/e2e: build/e2e
	test -z "$(shell git status --porcelain $(e2e_bindings_dest)/*)" || (git diff; false)

.PHONY: clean
clean:
	rm -rf build/ $(py_bindings_dest) $(ts_bindings_dest)

.PHONY: clean-all
clean-all: clean
