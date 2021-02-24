codegen_cli = swagger-codegen-cli-2.4.14.jar

.PHONY: all
all: get-deps
	$(MAKE) build

.PHONY: get-swagger-sepc
get-swagger-spec:
	cat $(DET_ROOT)/proto/build/swagger/determined/api/v1/api.swagger.json | jq > api.swagger.json

.PHONY: get-deps
get-deps:
	mkdir -p deps
	curl https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.14/$(codegen_cli) \
		-o deps/$(codegen_cli)

deps/codegen.stamp: deps/$(codegen_cli)

.PHONY: build-%
build-%:
	[ -f ./api.swagger.json ] || exit 1
	mkdir -p $*-bindings
	java -jar deps/$(codegen_cli) generate -i ./api.swagger.json \
		-l $* -o $*-bindings

.PHONY: build
build: build-typescript-fetch build-python build-go

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
