# syntax=docker/dockerfile:1
FROM node:16-alpine as builder

WORKDIR /build

# Copy over all source code
COPY . .

# Install build dependencies
RUN npm install

# Run build scripts for test scripts
RUN npm start

FROM grafana/k6:0.46.0 as base

WORKDIR /app

# Pull just the test script and module
COPY --from=builder \
	/build/build/api_performance_tests.js \
	/build/build/api_performance_tests.js.map \
	/app/

# When run, this should be overwritten to change the test target
ENV DET_MASTER="http://localhost:8080"
ENTRYPOINT ["k6"]
CMD ["run", "/app/api_performance_tests.js"]
