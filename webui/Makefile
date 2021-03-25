.PHONY: all
all:
	$(MAKE) get-deps
	$(MAKE) build

.PHONY: clean
clean: clean-react clean-tests
.PHONY: clean-%
clean-%:
	$(MAKE) -C $* clean

.PHONY: fmt
fmt: fmt-react
.PHONY: fmt-%
fmt-%:
	$(MAKE) -C $* fmt

.PHONY: check
check: check-react
.PHONY: check-%
check-%:
	$(MAKE) -C $* check

.PHONY: get-deps
get-deps:
	$(MAKE) -C react $@

.PHONY: build
build:
	$(MAKE) -C react build

.PHONY: test
test: test-react test-tests
.PHONY: test-%
test-%:
	$(MAKE) -C $* test
