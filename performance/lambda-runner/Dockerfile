# Dockerfile should be built from root of Performance.
FROM node:16-alpine as builder

WORKDIR /build

# cp ../master/build/determined-master master-binary
# mkdir -p master-root && cp -r ../master/static ./master-root/static
#
COPY . .
RUN npm install
RUN npm start

FROM grafana/k6:0.46.0 as base

FROM ubuntu:20.04 as ubuntu_stage

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get install -y python3-pip awscli && \
    pip install requests determined psycopg2-binary

COPY --from=base /usr/bin/k6 /usr/bin/k6
COPY --from=builder \
	/build/build/api_performance_tests.js \
	/build/build/api_performance_tests.js.map \
    /build/master-root \
    /build/lambda-runner/run.sh \
    /build/lambda-runner/master-config.yaml \
    /build/lambda-runner/wait_for_perf_migration_upload_results.py \
    /build/lambda-runner/upload_perf_results.py \
    /build/master-binary \
	/test_scripts/

ENTRYPOINT ["/test_scripts/run.sh"]
