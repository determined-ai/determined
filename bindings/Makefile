SWAGGER_SPEC := ../proto/build/swagger/determined/api/v1/api.swagger.json
BIN_VER := 2.4.21
CODEGEN_BIN := swagger-codegen-cli-$(BIN_VER).jar

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
	touch build/python-fastapi

build/%: $(SWAGGER_SPEC) deps/$(CODEGEN_BIN)
	mkdir -p build/$*
	java -jar deps/$(CODEGEN_BIN) generate -i $(SWAGGER_SPEC) \
		-l $* -o build/$*
	touch build/$*

.PHONY: place
place: build
	# process
	# fd -t f .py build/python | xargs sed 's/swagger_client/determined.common.api.swagger_client/' -i
	# rm -rf ../harness/determined/common/api/swagger_client
	# cp -r build/python/swagger_client ../harness/determined/common/api

	# sed -i '10s/.*/        if hasattr(model_class, "update_forward_refs"):/' \
	# 	build/python-fastapi/fastapi_client/__init__.py
	cp ./fastapi.override.py build/python-fastapi/fastapi_client/__init__.py
	cp ./models.override.py build/python-fastapi/fastapi_client/models.py # TODO TEMP remove
	rm -rf ../harness/determined/common/api/fastapi_client
	cp -r build/python-fastapi/fastapi_client ../harness/determined/common/api

.PHONY: build
build: build/typescript-fetch build/python-fastapi

.PHONY: clean-deps
clean-deps:
	rm -rf deps/

.PHONY: clean
clean:
	rm -rf build/

.PHONY: clean-all
clean-all: clean clean-deps
