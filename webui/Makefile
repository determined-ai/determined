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

get-deps:
	$(MAKE) -C api-ts-sdk $@
	$(MAKE) -C react $@
	$(MAKE) -C elm $@
	$(MAKE) -C tests $@

.PHONY: build
build: build-react build-elm

.PHONY: build-react
build-react: build-api-ts-sdk
	$(MAKE) -C react build

.PHONY: build-elm
build-elm:
	$(MAKE) -C elm build

.PHONY: build-api-ts-sdk
build-api-ts-sdk:
	$(MAKE) -C api-ts-sdk build

test:
	$(MAKE) -C react $@
	$(MAKE) -C elm $@
	$(MAKE) -C tests $@
