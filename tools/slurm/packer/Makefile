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
clean: 
	rm -rf build/

# Dfeaults to "slurm"
WORKLOAD_MANAGER ?= slurm	

.PHONY: build
build: main.pkr.hcl
	mkdir -p build/
	./scripts/generate-pkr-vars.sh $(WORKLOAD_MANAGER) > build/variables.pkrvars.hcl
	packer build -var-file="build/variables.pkrvars.hcl" main.pkr.hcl 
	WORKLOAD_MANAGER=$(WORKLOAD_MANAGER) ./scripts/update-variables.sh