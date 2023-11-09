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
	# Terraform controls the static cloudfront configuration.
	terraform apply -auto-approve
	# publish-to-s3.sh publishes the per-version content.
	./publish-to-s3.sh --version $(shell cat ../../VERSION)


.PHONY: check
check: init
	terraform fmt -check=true -diff=true
	terraform validate -check-variables=false
