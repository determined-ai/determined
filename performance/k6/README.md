# Performance Tests

## Nightly Runs

The code here is used to run performance tests nightly against the latest
published [determined master
image](https://hub.docker.com/r/determinedai/determined-master).

The automation for this is defined in a GitHub Actions workflow [in this
repo](../.github/workflows/performance-tests.yml). Reports from those runs will
be posted directly to our internal [#ci-bots Slack
channel](https://hpe-aiatscale.slack.com/archives/C9LFPNA3Y).

## Iterating

### Local Dev Requirements

Iterating on the performance test scripts or deployment code requires the
following CLIs:

- `npm`
- `make`
- `jq`
- `docker`
    - just the CLI is required, any backend will be fine to use (e.g. `colima`,
      `podman`, etc)
- `det`
    - this can be either built locally using `make build` in the repo root, or
      installed to a separate PyVenv using `pip install determined` for the
      latest version
- `aws`
    - the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables
      will be necessary for both this and the `det` CLI to use the `up`, `down`
      `persist`, and `unpersist` Make targets

### Performance Test Source

The performance test scripts live under the `src` directory, along with utility
scripts for modifying output and other common test tasks.

These can be compiled locally using `npm install` followed by `npm start`. This
will generate JS files for use in the `build` directory which can be referenced
by `k6`.

Below is a sample run of the performance tests compiled locally:

```bash
k6 run \
	-e DET_ADMIN_USERNAME=admin \
	-e DET_ADMIN_PASSWORD="" \
	-e metric_type=METRIC_TYPE_VALIDATION \
	-e batches=1000 \
	-e model_name='Test model' \
	-e model_version_number=1 \
	-e trial_id=1 \
	-e experiment_id=10951 \
	-e metric_name=validation_error \
	-e batches=100 \
	-e batches_margin=1000 \
	-e DET_MASTER=http://localhost:8080
	./build/api_performance_tests.js
```

For a local cluster to run tests against, use the `up-local` and `down-local`
`make` targets to spin up and tear down a local `det` cluster for performance
tests.

### Performance Test Image

The image code is contained entirely in the `Dockerfile`. When building, the
image will use one stage to compile the test scripts and then a second one to
bundle `k6` and the test scripts ready to run against a target cluster.

To iterate on the Docker image built to bundle our performance tests, use the
`build` and `run-local` targets to compile and execute a local-only build of the
image.

## Makefile

### Tasks

| Task Name | Description |
| --- | --- |
| `build` | Build docker image containing performance tests |
| `up` | Deploy a determined cluster in AWS ready to run tests against |
| `up-local` | Deploy a local cluster for iterating on performance tests |
| `down` | Tear-down a previously-deployed AWS cluster |
| `down-local` | Tear-down a previously-deployed local cluster |
| `persist` | Prevent redeploys and tear-downs of a remote AWS cluster |
| `depersist` | Reallow redeploys and tear-downs of a remote AWS cluster |
| `run` | Run the performance test suite against a remote AWS cluster |
| `run-local` | Run the performance test suite against a local cluster |
| `clean` | Remove intermediary files, including local performance test images |

### Vars

| Var Name | Description | Default |
| --- | --- | --- |
| `IMAGE_REPO` | Performance tests image repo name | `determinedai/perf-test` |
| `IMAGE_TAG` | Performance tests image tag, see [image](#image) | `${USER}` |
| `CLUSTER_ID` | Cluster ID for remote deployments | `${USER}-perf` |
| `KEYPAIR_ID` | AWS Keypair ID to use for remote deployments | `${USER}` |
| `VERSION` | Determined version to run on deployed clusters | Local `det -v` |
| `DET_URL` | Cluster API address to run tests against | Remote deployed cluster address |
| `DET_ADMIN_USERNAME` | Admin username to use in tests | `admin` |
| `DET_ADMIN_PASSWORD` | Admin password to use in tests | `""` |
