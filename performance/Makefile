# Docker image
IMAGE_REPO ?= determinedai/perf-test
IMAGE_TAG ?= $(USER)

# Perf testing cluster
CLUSTER_ID ?= $(USER)-perf
KEYPAIR_ID ?= performance-tests
VERSION ?= $(shell det -v | cut -d' ' -f2)

# Perf test options
DET_MASTER ?= $(shell cat .det.url || echo http://localhost:8080)
DET_ADMIN_USERNAME ?= admin
DET_ADMIN_PASSWORD ?= ""

IMAGE = $(IMAGE_REPO):$(IMAGE_TAG)
DET_FLAGS = \
		--cluster-id $(CLUSTER_ID) \
		--keypair $(KEYPAIR_ID) \
		--deployment-type simple-rds \
		--det-version $(VERSION) \
		--yes
K6_FLAGS = \
		-e DET_ADMIN_USERNAME=$(DET_ADMIN_USERNAME) \
		-e DET_ADMIN_PASSWORD=$(DET_ADMIN_PASSWORD)
PERSIST_CHECK = \
	PERSIST_VALUE=$$(\
		aws cloudformation describe-stacks --stack-name $(CLUSTER_ID) | \
		jq -r '.Stacks[0].Tags[] | select(.Key == "persist").Value' \
	); \
	if [ "$$PERSIST_VALUE" = "true" ]; then \
		echo 'Cluster currently persisted'; \
		exit 1; \
	else \
		echo 'Cluster not currently persisted'; \
	fi


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
	det deploy aws down --cluster-id $(CLUSTER_ID) --yes && \
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
	docker run \
		--name perf-tests ${PERF_DOCKER_FLAGS} \
		$(IMAGE) run \
		-e DET_MASTER=$(DET_MASTER) \
		$(K6_FLAGS) ${PERF_K6_FLAGS} \
		/test_scripts/api_performance_tests.js --summary-export=summary.json; \
	TEST_STATUS=$$?; \
	docker cp \
		perf-tests:/home/k6/junit.xml \
		reports/$$(date +%s).junit.xml; \
	docker cp \
		perf-tests:/home/k6/results.txt \
		reports/$$(date +%s).results.txt; \
	docker cp \
		perf-tests:/home/k6/summary.json \
		reports/latest.results.json; \
	docker rm perf-tests; \
	[ $$TEST_STATUS -eq 0 ] || exit $$TEST_STATUS

clean:
	docker image rm $(IMAGE)
	rm -f .up.out .det.url
