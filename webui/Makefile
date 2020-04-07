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
	$(MAKE) -C react $@
	$(MAKE) -C elm $@

build:
	$(MAKE) -C react $@
	$(MAKE) -C elm $@

test:
	$(MAKE) -C react $@
	$(MAKE) -C elm $@
	$(MAKE) -C tests get-deps $@
