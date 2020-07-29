PROJECT_ROOT := "../.."

.PHONY: all
all: get-deps build


.PHONY: get-deps-swagger
get-deps-swagger:
	mkdir -p deps
	curl https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.14/swagger-codegen-cli-2.4.14.jar \
		-o deps/swagger-codegen-cli.jar

.PHONY: get-deps-package
get-deps-package:
	npm ci --no-audit

.PHONY: build-swagger
build-swagger:
	mkdir -p src
	java -jar deps/swagger-codegen-cli.jar generate -i \
		${PROJECT_ROOT}/proto/build/swagger/determined/api/v1/api.swagger.json \
		-l typescript-fetch -o src/

.PHONY: build-package
build-package:
	# depends on build-swagger
	npm run build

.PHONY: get-deps
get-deps: get-deps-swagger get-deps-package

.PHONY: build
build: build-swagger
	$(MAKE) build-package

.PHONY: clean
clean:
	rm -rf deps/ node_modules/ src/ lib/
