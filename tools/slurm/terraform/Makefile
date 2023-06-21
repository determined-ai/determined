# Commands for managing (creating/destroying/etc) resources.

build/backend.conf: scripts/generate-backendconf.sh
	mkdir -p build
	./scripts/generate-backendconf.sh > build/backend.conf

.PHONY: init
init: build/backend.conf
	terraform init -backend-config build/backend.conf

.PHONY: fmt
fmt:
	terraform fmt

.PHONY: check
check:
	terraform validate

build/generate-sshkeys: scripts/generate-sshkeys.sh
	mkdir -p build
	./scripts/generate-sshkeys.sh
	@touch build/generate-sshkeys

build/default.tfvars: scripts/generate-tfvars.sh build/generate-sshkeys
	mkdir -p build
	./scripts/generate-tfvars.sh $(VMTIME) > build/default.tfvars

.PHONY: build
build: init build/default.tfvars
	terraform apply --var-file build/default.tfvars

.PHONY: clean
clean: build/default.tfvars
	terraform destroy --var-file build/default.tfvars
	rm -rf build

# Commands for interacting with created resources.

ZONE = $(shell terraform output --raw zone)
INSTANCE_NAME = $(shell terraform output --raw instance_name)
PROJECT = $(shell terraform output --raw project)
.PHONY: connect
connect:
	gcloud compute ssh --zone "$(ZONE)" "$(INSTANCE_NAME)" --project "$(PROJECT)"

.PHONY: tunnel
tunnel:
	gcloud compute ssh --zone "$(ZONE)" "$(INSTANCE_NAME)" --project "$(PROJECT)" \
		-- -L 8081:localhost:8081 -R 8080:localhost:8080

