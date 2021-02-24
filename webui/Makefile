.PHONY: all
all:
	$(MAKE) get-deps
	$(MAKE) build

.PHONY: clean
clean: clean-react clean-tests clean-api-bindings
.PHONY: clean-%
clean-%:
	$(MAKE) -C $* clean

.PHONY: fmt
fmt: fmt-react fmt-tests
.PHONY: fmt-%
fmt-%:
	$(MAKE) -C $* fmt

.PHONY: check
check: check-react check-tests
.PHONY: check-%
check-%:
	$(MAKE) -C $* check

.PHONY: get-deps
get-deps: build-api-bindings
	$(MAKE) -C react $@
	$(MAKE) -C tests $@

.PHONY: build
build: build-react
.PHONY: build-proto
build-proto:
	$(MAKE) -C ../proto build

.PHONY: build-api-bindings
build-api-bindings: build-proto
	$(MAKE) -C ../proto/api-bindings all
.PHONY: build-%
build-%:
	$(MAKE) -C $* build

.PHONY: test
test: test-react test-tests
.PHONY: test-%
test-%:
	$(MAKE) -C $* test
