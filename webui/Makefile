.PHONY: default get-deps build clean test fmt check

default: get-deps build

clean: clean-react clean-elm clean-tests
	$(MAKE) -C api-ts-sdk $@
.PHONY: clean-%
clean-%:
	$(MAKE) -C $(subst -,/,$*) clean

.PHONY: fmt
fmt: fmt-react fmt-elm fmt-tests
.PHONY: fmt-%
fmt-%:
	$(MAKE) -C $(subst -,/,$*) fmt

.PHONY: check
check: check-react check-elm check-tests
.PHONY: check-%
check-%:
	$(MAKE) -C $(subst -,/,$*) check

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
	$(MAKE) -C api-ts-sdk get-deps build

.PHONY: build-react
build-react:
	$(MAKE) -C react build

.PHONY: build-elm
build-elm:
	$(MAKE) -C elm build

.PHONY: test
test: test-react test-elm test-tests
.PHONY: test-%
test-%:
	$(MAKE) -C $(subst -,/,$*) test
