## Quick start

1. Install Terraform following [these instructions](https://developer.hashicorp.com/terraform/downloads).
2. Download the [GCP CLI](https://cloud.google.com/sdk/docs/install-sdk) and run `gcloud auth application-default login` to get credentials.
3. Run `make slurmcluster` from the root of the repo and wait (up to 10 minutes) for it to start.
   1. To specify which container run time environment to use, pass in the `cont=[container_run_type]` to `make slurmcluster`. Choose from either `singularity` (default), `podman`, or `enroot`.
4. Step 2 will ultimately launch a local devcluster. Use this as you typically would [1].
5. Release the resources with `make unslurmcluster` when you are done.

[1] It is fine to exit this and restart it with `make slurmcluster` again as you please.

To see usage of the `make slurmcluster` target, run `make -C tools/slurm usage`.

## Alternatives

The `make slurmcluster` flow is fast and convenient. Alternatively, if you use
`make -C terraform build` and then just use the resulting instance as a dev box after 
installing Determined and VS Code Remote on it, the experience is better after getting it
setup (though this somewhat is a matter of preference).

## `make slurmcluster` notes

To run Determined + Slurm, you have a few options:

- Spin up a Linux development machine, install all the prerequisite software and run a cluster as a
  customer would, following our publically available documentation.
- Use `tools/slurmcluster.sh` by following the usage documentation for the script after getting
  access to one of the systems it supports.
- Or use `make slurmcluster` (the code contained within this directory and its children).

Under the hood, this launches a compute instance with Slurm, Singularity (Apptainer), Podman, Enroot, the Cray
Launcher component and many other dependencies pre-installed. Then, SSH tunnels are opened so that
`localhost:8081` on your machine points at port `8081` on compute instance and
port `8081` on the compute instance points at `localhost:8080` on your machine. Last,
`devcluster` is started with the Slurm RM pointed at the remote instance, and local development
with `devcluster` works from here as always.

# Running pytest Suites

## In Development

To locally invoke pytests that run on your slurmcluster,
  1.  Run `make slurmcluster cont=[container_run_type]` and wait for the `devcluster` to spin up.
  2.  Set `export DET_MASTER="http://localhost:8080"` *or* run `unset DET_MASTER` (since `make slurmcluster` listens on `localhost:8080` by default). **Note**: this is *different* than the master port when running `tools/slurmcluster.sh` (8081). Be careful to set/unset accordingly.
  3.  Login as `determined` by running `det user login determined`. **Note**: `make slurmcluster` automatically links your user account with the agent.
  4.  `cd` to `determined-ee/e2e_tests`.
  5.  Run the following command which will run all tests with the `e2e_slurm` mark that *do not also* have the `parallel` mark. **Note**: The *pytest* command invocation for each `container_run_type` is the same.
```
  pytest --capture=tee-sys -vv \                                                                                                 
-m 'e2e_slurm and not parallel' \
--durations=0 \
--master-scheme="http" \
--master-host="localhost" \
--master-port="8080" \
-k 'not start_and_write_to_shell'
```

## On CircleCI

Upon each commit and push, CircleCI invokes three test suites: `test-e2e-singularity-gcp`, `test-e2e-podman-gcp`, and `test-e2e-enroot-gcp`. Each of these CircleCI jobs actually invoke the same exact tests, only in different container runtime environments (see section 3.1 of [Quick start](#quick-start)). The pytests on CircleCI are invoked by the following command:

```
pytest --capture=tee-sys -vv \
-m 'e2e_slurm and not parallel' \
--durations=0 \
--master-scheme="http" \
--master-host="localhost" \
--master-port="8080" \
-o junit_family=xunit1 \
--junit-xml="/tmp/test-results/e2e/tests.xml" \
-k 'not start_and_write_to_shell' \
```
This invocation specifies that all tests are to be run via the remote launcher running on `localhost:8080`. The command specifies that all pytests with the `e2e_slurm` mark should be run except those that have the `parallel` mark as well (this is due to the fact that there are no GPUs and only 1 node on the GCP compute instances). The `-k 'not start_and_write_to_shell'` specifies to not run the `start_and_write_to_shell` function in `e2e_tests/tests/command/test_shell.py`. This test is currently skipped due to a proxy issue on GCP.

## Notes on `make slurmcluster` tests on CircleCI 

The following test suites currently run only on hardware. They do not run successfully with `make slurmcluster` and thus are not executed via GCP as part of the CI/CD gate:
  - `test-e2e-slurm-gpu`: Test is skipped because the compute instance that the tests run on do not have any GPUs.
  - `test-e2e-slurm-misconfigured`: This test could be made to work, but requires passing in a misconfigured `master.yaml` to the launcher on GCP, which could be tedious.
  - `test-e2e-slurm-preemption-quarantine`: Currently runs on znode as a part of the nightly test suite.
  - `test-e2e-slurm-restart`: Dependent upon znode configuration, so not worth testing on GCP.

