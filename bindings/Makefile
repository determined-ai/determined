PROJECT_ROOT := ..
SWAGGER_SPEC := ${PROJECT_ROOT}/proto/build/swagger/determined/api/v1/api.swagger.json
CODEGEN_BIN := swagger-codegen-cli-2.4.14.jar

.PHONY: all
all: get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: deps/${CODEGEN_BIN}

deps/${CODEGEN_BIN}:
	mkdir -p deps
	curl https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.14/${CODEGEN_BIN} \
		-o deps/${CODEGEN_BIN}

build/%: ${SWAGGER_SPEC}
	[ -f ./${SWAGGER_SPEC} ] || exit 1
	mkdir -p build/$*
	java -jar deps/$(CODEGEN_BIN) generate -i ${SWAGGER_SPEC} \
		-l $* -o build/$*
	touch build/$*

.PHONY: build-default
build-default: build/typescript-fetch build/python

.PHONY: build
build: ${SWAGGER_SPEC}
	$(MAKE)	build-default

.PHONY: setup-example-%
setup-example-%:
	cp -r examples/$*/* build/$*/

.PHONY: clean-deps
clean-deps:
	rm -rf deps/

.PHONY: clean
clean:
	rm -rf build/

.PHONY: clean-all
clean-all: clean clean-deps
