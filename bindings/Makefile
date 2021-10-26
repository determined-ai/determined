SWAGGER_SPEC := ../proto/build/swagger/determined/api/v1/api.swagger.json
BIN_VER := 2.4.21
CODEGEN_BIN := swagger-codegen-cli-$(BIN_VER).jar
PYAPI_OUT_DIR := ../harness/determined/common/api/fastapi_client

.PHONY: all
all: get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: deps/$(CODEGEN_BIN)
	rm -rf deps/fastapi_client
	cd deps; git clone https://github.com/dmontagu/fastapi_client.git

deps/$(CODEGEN_BIN):
	mkdir -p deps
	curl https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/$(BIN_VER)/$(CODEGEN_BIN) \
		-o deps/$(CODEGEN_BIN)

build/python-fastapi: $(SWAGGER_SPEC) deps/$(CODEGEN_BIN)
	deps/fastapi_client/scripts/generate.sh --import-name determined.common.api.fastapi_client \
		--package-name fastapi_client \
		-i ~/projects/da/determined/proto/build/swagger/determined/api/v1/api.swagger.json \
		-o build/python-fastapi
	# remove unused files
	touch build/python-fastapi
	$(MAKE) fapi-postprocess

.PHONY: fapi-postprocess
fapi-postprocess:
	rm -r build/python-fastapi/fastapi_client/docs/ || exit 0
	rm build/python-fastapi/fastapi_client/api_client.py || exit 0
	rm build/python-fastapi/fastapi_client/exceptions.py || exit 0
	rm -rf $(PYAPI_OUT_DIR) || exit 0
	cp -r build/python-fastapi/fastapi_client $(PYAPI_OUT_DIR)
	cp ./fastapi.override.py $(PYAPI_OUT_DIR)/__init__.py
	fd -t f api $(PYAPI_OUT_DIR)/api | xargs sed 's/^.*fastapi.*jsonable.*/from determined.common.api.fapi import to_jsonable as jsonable_encoder/' -i
	fd -t f api $(PYAPI_OUT_DIR)/api | xargs sed 's/^.*fastapi_client.api_cli.*/    from determined.common.api.fapi import ApiClient/' -i
	sed 's/^from.*Field/from determined.common.api.fapi import BaseModel, Field/' -i $(PYAPI_OUT_DIR)/models.py

build/%: $(SWAGGER_SPEC) deps/$(CODEGEN_BIN)
	mkdir -p build/$*
	java -jar deps/$(CODEGEN_BIN) generate -i $(SWAGGER_SPEC) \
		-l $* -o build/$*
	touch build/$*

.PHONY: build
build: build/typescript-fetch

.PHONY: clean-deps
clean-deps:
	rm -rf deps/

.PHONY: clean
clean:
	rm -rf build/

.PHONY: clean-all
clean-all: clean clean-deps
