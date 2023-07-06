# Starting up a slurmcluster on GCP

## Quick start

1. Install Terraform following [these instructions](https://developer.hashicorp.com/terraform/downloads).
2. Download the [GCP CLI](https://cloud.google.com/sdk/docs/install-sdk) and run `gcloud auth application-default login` to get credentials.
3. Run `make slurmcluster` from the root of the repo and wait (up to 10 minutes) for it to start.
   - To specify which container run time environment to use, pass in `flags="-c {container_run_type}"` to `make slurmcluster`. Choose from either `singularity` (default), `podman`, or `enroot`.
   - To specify which workload manager to use, pass in `flags="-w {workload_manager}"` to `make slurmcluster`. Choose from either `slurm` (default) or `pbs`. Note: in specifying the workload manager, `make slurmcluster` will automatically load the appropriate boot disk image (found in `terraform/images.conf`).
   - By default, all VMs created with `make slurmcluster` will be destroyed after 7200 seconds (2 hours). To sepcify a different amount of time, pass in `flags="-t {time_seconds}"` to `make slurmcluster`.
4. Step 2 will ultimately launch a local devcluster. Use this as you typically would [1].
5. Release the resources with `make unslurmcluster` when you are done.

Once the VM is created and SSH connection is established, one can also directly connect to the created instance with the command 
```
gcloud compute ssh --zone <zone_name> <instance_name> --project <project_name>
```

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

## Automatic VM Deletion

By default, each devbox invoked by `make slurmcluster` will automatically delete the VM after two hours of runtime. If you want to override this time limit, one can run `make slurm cluster vmtime=[seconds]`. Where `seconds` is a value between 0 to 315,576,000,000 seconds inclusive. The two hour time limit ensures that devboxes are being deleted if they are not used to prevent excess costs.

# Running pytest Suites

## In Development

To locally invoke pytests that run on your slurmcluster,
  1.  Run `make slurmcluster [flags="options"]` and wait for the `devcluster` to spin up.
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
-k 'not cifar10_pytorch_distributed'
```

## On CircleCI

Upon each commit and push, CircleCI invokes three test suites: `test-e2e-singularity-gcp`, `test-e2e-podman-gcp`, and `test-e2e-enroot-gcp`. Each of these CircleCI jobs actually invoke the same exact tests, only in different container runtime environments (see section 3.1 of [Quick start](#quick-start)). The pytests on CircleCI are invoked by the following command [2]:

```
pytest --capture=tee-sys -vv \
-m 'e2e_slurm and not parallel' \
--durations=0 \
--master-scheme="http" \
--master-host="localhost" \
--master-port="8080" \
-o junit_family=xunit1 \
--junit-xml="/tmp/test-results/e2e/tests.xml" \
-k 'not cifar10_pytorch_distributed'
```
[2]: When invoking `make slurmcluster flags="-w pbs"`, append `and not test_docker_image and not test_bad_slurm_option and not test_launch_layer_exit` to the end of the existing `-k` argument string. These tests do not currently run on PBS instances.

This invocation specifies that all tests are to be run via the remote launcher running on `localhost:8080`. The command specifies that all pytests with the `e2e_slurm` mark should be run except those that have the `parallel` mark as well (this is due to the fact that there are no GPUs and only 1 node on the GCP compute instances). The `and not cifar10_pytorch_distributed` specifies also not to run the `cifar10_pytorch_distributed` test. This test is omitted because it takes too long to run on instances with only one node.

# Notes on `make slurmcluster` tests on CircleCI 

By default, the `test-e2e-*-gcp` jobs are not run within the `test-e2e` workflow on a **developer branch**. If you would like to invoke these jobs on a certain commit, you must add the "[ALLGCP]" keyword to the commit message. For example,
```
git add --all
git commit -m "[ALLGCP] This is my commit where all hpc-gcp jobs will run."
git push
```
will invoke the slurm-gcp jobs within the `test-e2e` workflow.
**On branch `main` and `release/rc` branches, these jobs always run, regardless of commit message.**

The following test suites currently run only on hardware. They do not run successfully with `make slurmcluster` and thus are not executed via GCP as part of the CI/CD gate:
  - `test-e2e-slurm-gpu`: Test is skipped because the compute instance that the tests run on do not have any GPUs.
  - `test-e2e-slurm-misconfigured`: This test could be made to work, but requires passing in a misconfigured `master.yaml` to the launcher on GCP, which could be tedious.
  - `test-e2e-slurm-preemption-quarantine`: Currently runs on znode as a part of the nightly test suite.
  - `test-e2e-slurm-restart`: Dependent upon znode configuration, so not worth testing on GCP.

## Important Workaround Explained

Recall, a CircleCI pipeline is triggered upon a push to the remote repository. Only one pipeline can be running at once on a single branch *unless* that branch is main. If a pipeline is in progress and other changes are pushed to a developer branch, the running workflow will be **canceled**. This is a problem if the build was canceled in between the `make slurmcluster` and `make unslurmcluster` steps because then the VM and its current state (all of its experiments, queued jobs, processes, etc.) will persist to the *subsequent* pipeline on the developer branch.

There are different aspects to the issue which are addressed in various ways, as follows. 

1. Making a completely separate VM for *each distinct* job in *each distinct* workflow would allow too many opportunities for VMs and VPC networks to become stale (zombie state). Therefore, one VM is created for each distinct `GH username` + `GH branch` + `CircleCI job name` combination. This way, if a user cancels a build in the "danger zone" (between `make/unmake slurmcluster`) then the VM will not become stale forever and can be dealt with on a subsequent push.
2. However, on said subsequent pushes, the state of the VM will persist (as mentioned before) which will cause conflicts with launching experiments and starting a `devcluster`. Therefore, we must **delete** the persistent VM *if it exists* by executing `gcloud compute instances delete <relevant instance>`. You will notice in `config.yaml` that the zone is specified via a complicated `grep` command. This command grabs the **default** `zone` value from `terraform/variables.tf` and specifies it in the aforementioned command. This is done since we are running this deletion command *before* we get the chance to reach `make slurmcluster`, which assigns the correct zone to `GOOGLE_COMPUTE_ZONE`. Evidently, this workaround is paradoxical in and of itself.
3. This deleted instance mentioned above will have acquired a state lock on the terraform states, but now it is deleted and will never give up the lock. Thus, the final step of this workaround is to pass in the `tf_lock=false` environment variable to `make slurmcluster` which runs `terraform apply/destroy` with the flag `-lock=false`. This tells terraform to ignore the status of the lock which allows us to `make/unmake slurmcluster`. Keep in mind that this would normally be dangerous, however since the instance is deleted in the previous step, there will never be any contention in changing the terraform states.

The issue of CircleCI not having a way to gracefully terminate a workflow upon cancellation is a [known issue](https://discuss.circleci.com/t/cancel-workflow-job-with-graceful-termination/39172). If CircleCI ever provides this feature, this workaround can be deprecated and one could simply add `make unslurmcluster` to a cleanup script that runs upon cancellation.

