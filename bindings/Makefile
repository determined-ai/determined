SWAGGER_SPEC := ../proto/build/swagger/determined/api/v1/api.swagger.json
CODEGEN_BIN := swagger-codegen-cli-2.4.14.jar
CLIENT_PATH := ../harness/determined/_swagger/client

.PHONY: all
all: get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: deps/$(CODEGEN_BIN)

deps/$(CODEGEN_BIN):
	mkdir -p deps
	curl https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.14/$(CODEGEN_BIN) \
		-o deps/$(CODEGEN_BIN)

.PHONY: build-typescript-fetch
build-typescript-fetch: $(SWAGGER_SPEC) deps/$(CODEGEN_BIN)
	mkdir -p build/typescript-fetch
	java -jar deps/$(CODEGEN_BIN) generate -i $(SWAGGER_SPEC) \
		-l typescript-fetch -o build/typescript-fetch
	touch build/typescript-fetch

.PHONY: build-python
build-python: $(SWAGGER_SPEC) deps/$(CODEGEN_BIN)
	mkdir -p build/python
	java -jar deps/$(CODEGEN_BIN) generate -i $(SWAGGER_SPEC) \
		-l python -o build/python -c config.json
	touch build/python
	rm -rf $(CLIENT_PATH)
	mkdir -p $(CLIENT_PATH)/api
	mkdir -p $(CLIENT_PATH)/models
	cp -R ./build/python/determined._swagger.client/. $(CLIENT_PATH)/.
	cp -R ./build/python/determined/_swagger/client/. $(CLIENT_PATH)/.

.PHONY: build
build: build-typescript-fetch build-python

.PHONY: check
check: clean
	make -C ../proto
	make build-python
	test -z "$(shell git status --porcelain '$(CLIENT_PATH)')"

.PHONY: clean-deps
clean-deps:
	rm -rf deps/

.PHONY: clean
clean:
	rm -rf build/

.PHONY: clean-all
clean-all: clean clean-deps
