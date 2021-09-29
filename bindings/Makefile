SWAGGER_SPEC := ../proto/build/swagger/determined/api/v1/api.swagger.json
BIN_VER := 2.4.21
CODEGEN_BIN := swagger-codegen-cli-$(BIN_VER).jar

.PHONY: all
all: get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: deps/$(CODEGEN_BIN)

deps/$(CODEGEN_BIN):
	mkdir -p deps
	curl https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/$(BIN_VER)/$(CODEGEN_BIN) \
		-o deps/$(CODEGEN_BIN)

.PHONY: build/%
build/%: $(SWAGGER_SPEC) deps/$(CODEGEN_BIN)
	mkdir -p build/$*
	java -jar deps/$(CODEGEN_BIN) generate -i $(SWAGGER_SPEC) \
		-l $* -o build/$*
	touch build/$*

.PHONY: place
place: build
	# process
	fd -t f .py build/python | xargs sed 's/swagger_client/determined.common.api.swagger_client/' -i
	rm -rf ../harness/determined/common/api/swagger_client
	cp -r build/python/swagger_client ../harness/determined/common/api

.PHONY: build
build: build/typescript-fetch build/python

.PHONY: clean-deps
clean-deps:
	rm -rf deps/

.PHONY: clean
clean:
	rm -rf build/

.PHONY: clean-all
clean-all: clean clean-deps
