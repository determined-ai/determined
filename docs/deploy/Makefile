export TF_VAR_det_version := $(shell ../../version.sh)
export DET_VARIANT ?= OSS
export TF_VAR_det_variant := ${DET_VARIANT}
export TF_CLI_ARGS_init :=  "-backend-config=backend_${DET_VARIANT}.conf"

.PHONY: clean
clean:
	-rm -r .terraform
	-rm state/terraform.tfstate.backup

.PHONY: verify
verify: init
	terraform validate

.PHONY: init
init:
	terraform init -upgrade -reconfigure

.PHONY: plan
plan: init
	terraform plan

.PHONY: publish
publish: init
	terraform apply -auto-approve

.PHONY: check
check: init
	terraform fmt -check=true -diff=true
	$(MAKE) terraform verify
