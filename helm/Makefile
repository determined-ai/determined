fmt:
	helm lint charts/determined/

.PHONY: build
build: clean
	mkdir -p build/
	helm package --destination build charts/determined

.PHONY: clean
clean:
	rm -rf build/

.PHONY: release
release: export NGC_REPO := https://helm.ngc.nvidia.com/isv-ngc-partner/determined
release: export NGC_API_USERNAME ?=
release: export NGC_API_KEY ?=
release:
	helm repo add determined $$NGC_REPO --username $$NGC_API_USERNAME --password $$NGC_API_KEY
	helm push -f build/determined-0.4.0.tgz determined
