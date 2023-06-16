.PHONY: init
init:
	packer init .

.PHONY: fmt
fmt:
	packer fmt .

.PHONY: check
check:
	packer fmt -check .

.PHONY: build-debug
build-debug: main.pkr.hcl
	PACKER_LOG=10 packer build -debug main.pkr.hcl

.PHONY: clean
clean: rm -rf build/

.PHONY: build
build: main.pkr.hcl
	mkdir -p build/
	packer build -var='launcher_deb_name=$(shell ./scripts/get-launcher-image.sh)' main.pkr.hcl 
	./scripts/update-variables.sh