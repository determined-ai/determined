# Docker image
IMAGE_REPO ?= determinedai/perf-test
IMAGE_TAG ?= $(USER)

# Perf testing cluster
CLUSTER_ID ?= $(USER)-perf
KEYPAIR_ID ?= $(USER)
VERSION ?= $(shell det -v | cut -d' ' -f2)

# Perf test options
DET_URL ?= $(shell cat .det.url || echo UNSET)
DET_ADMIN_USERNAME ?= admin
DET_ADMIN_PASSWORD ?= ""

DET_FLAGS = \
		--cluster-id $(CLUSTER_ID) \
		--keypair $(KEYPAIR_ID) \
		--deployment-type simple-rds \
		--det-version $(VERSION) \
		--yes
IMAGE = $(IMAGE_REPO):$(IMAGE_TAG)
PERSIST_CHECK = \
	PERSIST_VALUE=$$(\
		aws cloudformation describe-stacks --stack-name $(CLUSTER_ID) | \
		jq -r '.Stacks[0].Tags[] | select(.Key == "persist").Value' \
	); \
	[[ $$PERSIST_VALUE == 'true' ]] && \
		echo 'Cluster currently persisted' && \
		exit 1 || \
		echo 'Cluster not currently persisted'


build: Dockerfile src/* package-lock.json package.json tsconfig.json webpack.config.js
	docker build . -t $(IMAGE)

up:
	$(PERSIST_CHECK)
	det deploy aws up \
		$(DET_FLAGS) \
		--add-tag persist=false \
	| tee .up.out && \
	cat .up.out \
		| grep DET_MASTER \
		| sed -rn 's/.*DET_MASTER=(.*)/\1/p' \
		> .det.url

up-local:
	det deploy local cluster-up \
		--cluster-name performance \
		--det-version $(VERSION) \
		--yes

down:
	$(PERSIST_CHECK)
	det deploy aws down --cluster-id $(CLUSTER_ID) && \
	rm -f .det.url

down-local:
	det deploy local cluster-down --cluster-name performance

persist:
	det deploy aws up \
		$(DET_FLAGS) \
		--add-tag persist=true

unpersist:
	det deploy aws up \
		$(DET_FLAGS) \
		--add-tag persist=false

run:
	touch junit.xml
	docker run \
		-v $(PWD)/junit.xml:/app/junit.xml \
		$(IMAGE) run \
		-e DET_MASTER=$(DET_URL) \
		-e DET_ADMIN_USERNAME=$(DET_ADMIN_USERNAME) \
		-e DET_ADMIN_PASSWORD=$(DET_ADMIN_PASSWORD) \
		/app/api_performance_tests.js
	mv junit.xml reports/$$(date +%s).junit.xml

run-local:
	touch junit.xml
	docker run \
		-v $(PWD)/junit.xml:/app/junit.xml \
		$(IMAGE) run \
		-e DET_MASTER=http://localhost:8080 \
		-e DET_ADMIN_USERNAME=$(DET_ADMIN_USERNAME) \
		-e DET_ADMIN_PASSWORD=$(DET_ADMIN_PASSWORD) \
		/app/api_performance_tests.js
	mv junit.xml reports/local-$$(date +%s).junit.xml

clean:
	docker image rm $(IMAGE)
	rm -f .up.out .det.url
