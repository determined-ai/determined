.PHONY: default get-deps build clean test fmt check

default: get-deps build

clean:
	$(MAKE) -C react $@
	$(MAKE) -C elm $@
	$(MAKE) -C tests $@

fmt:
	$(MAKE) -C react $@
	$(MAKE) -C elm $@
	$(MAKE) -C tests $@

check:
	$(MAKE) -C react $@
	$(MAKE) -C elm $@
	$(MAKE) -C tests $@

get-deps: build-api-ts-sdk
	$(MAKE) -C react $@
	$(MAKE) -C elm $@
	$(MAKE) -C tests $@

.PHONY: build
build: build-react build-elm

.PHONY: build-proto
build-proto:
	$(MAKE) -C ../proto build

.PHONY: build-api-ts-sdk
build-api-ts-sdk: build-proto
	$(MAKE) -C api-ts-sdk build

.PHONY: build-react
build-react: build-api-ts-sdk
	$(MAKE) -C react build

.PHONY: build-elm
build-elm:
	$(MAKE) -C elm build

test: build-api-ts-sdk
	$(MAKE) -C react $@
	$(MAKE) -C elm $@
	$(MAKE) -C tests $@
