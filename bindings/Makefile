PROJECT_ROOT := ".."
CODEGEN_BIN := swagger-codegen-cli-2.4.14.jar

.PHONY: all
all: get-deps get-swagger-spec
	$(MAKE) build

.PHONY: get-swagger-sepc
get-swagger-spec:
	$(MAKE) -C ${PROJECT_ROOT}/proto build
	cp ${PROJECT_ROOT}/proto/build/swagger/determined/api/v1/api.swagger.json .

.PHONY: get-deps
get-deps: deps/${CODEGEN_BIN}

deps/${CODEGEN_BIN}:
	mkdir -p deps
	curl https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.14/${CODEGEN_BIN} \
		-o deps/${CODEGEN_BIN}

.PHONY: build-%
build-%:
	[ -f ./api.swagger.json ] || exit 1
	mkdir -p $*-bindings
	java -jar deps/$(CODEGEN_BIN) generate -i ./api.swagger.json \
		-l $* -o $*-bindings


.PHONY: build-default
build-default: build-typescript-fetch build-python

.PHONY: build
build: get-swagger-spec
	$(MAKE)	build-default

.PHONY: setup-example-%
setup-example-%:
	cp -r examples/$*/* $*-bindings/

.PHONY: clean-deps
clean-deps:
	rm -rf deps/

.PHONY: clean
clean:
	rm -rf *-bindings

.PHONY: clean-all
clean-all: clean clean-deps
