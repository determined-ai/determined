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

.PHONY: build
build: main.pkr.hcl
	packer build main.pkr.hcl
