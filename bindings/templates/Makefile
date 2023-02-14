MAVEN_VER = 3.9.0
MAVEN_PATH := apache-maven-${MAVEN_VER}/bin/mvn
TEMPLATES_JAR := ts-globals-codegen-class.jar

.PHONY: clean-deps
clean-deps:
	rm -rf deps

.PHONY: all
all:
	$(MAKE) get-deps
	$(MAKE) build

.PHONY: get-deps
get-deps: deps/$(MAVEN_PATH)

deps/$(MAVEN_PATH):
	mkdir -p deps
	curl https://dlcdn.apache.org/maven/maven-3/$(MAVEN_VER)/binaries/apache-maven-$(MAVEN_VER)-bin.tar.gz \
		-o deps/maven.tar.gz
	tar -xvf deps/maven.tar.gz -C deps

.PHONY: build
build: build/$(TEMPLATES_JAR)

build/$(TEMPLATES_JAR): pom.xml $(wildcard src/**/*)  deps/$(MAVEN_PATH)
	deps/$(MAVEN_PATH) -f pom.xml package

.PHONY: clean
clean:
	rm -rf build
