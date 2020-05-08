export TF_VAR_det_version := $(shell cat ../../VERSION)

.PHONY: clean
clean:
	-rm -r .terraform
	-rm state/terraform.tfstate.backup

.PHONY: verify
verify:
	terraform --version

.PHONY: init
init:
	terraform init

.PHONY: plan
plan: init
	terraform plan

.PHONY: publish
publish: init
	terraform apply -auto-approve

.PHONY: check
check: init
	terraform fmt -check=true -diff=true
	terraform validate -check-variables=false
