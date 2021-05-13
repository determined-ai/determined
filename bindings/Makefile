SWAGGER_SPEC := ../proto/build/swagger/determined/api/v1/api.swagger.json
CODEGEN_BIN := swagger-codegen-cli-2.4.14.jar
CLIENT_PATH := ../harness/determined/swagger/client

.PHONY: all
all: get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: deps/$(CODEGEN_BIN)

deps/$(CODEGEN_BIN):
	mkdir -p deps
	curl https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.14/$(CODEGEN_BIN) \
		-o deps/$(CODEGEN_BIN)

build/%: $(SWAGGER_SPEC) deps/$(CODEGEN_BIN)
	mkdir -p build/$*
	java -jar deps/$(CODEGEN_BIN) generate -i $(SWAGGER_SPEC) \
		-l $* -o build/$* -c config.json
	touch build/$*

.PHONY: build
build: build/typescript-fetch build/python
	cp -rf ./build/python/determined.swagger.client/ ./build/python/determined/swagger/client
	cp -rf ./build/python/determined/swagger/client/ $(CLIENT_PATH)
	test -z "$(shell git status --porcelain '$(CLIENT_PATH)')"

.PHONY: clean-deps
clean-deps:
	rm -rf deps/

.PHONY: clean
clean:
	rm -rf build/

.PHONY: clean-all
clean-all: clean clean-deps
