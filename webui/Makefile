.PHONY: all
all:
	$(MAKE) get-deps
	$(MAKE) build

.PHONY: clean
clean: clean-react clean-elm clean-tests clean-api-ts-sdk
.PHONY: clean-%
clean-%:
	$(MAKE) -C $* clean

.PHONY: fmt
fmt: fmt-react fmt-elm fmt-tests
.PHONY: fmt-%
fmt-%:
	$(MAKE) -C $* fmt

.PHONY: check
check: check-react check-elm check-tests
.PHONY: check-%
check-%:
	$(MAKE) -C $* check

.PHONY: get-deps
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
	$(MAKE) -C api-ts-sdk get-deps
	$(MAKE) -C api-ts-sdk build
.PHONY: build-%
build-%:
	$(MAKE) -C $* build

.PHONY: test
test: test-react test-elm test-tests
.PHONY: test-%
test-%:
	$(MAKE) -C $* test
