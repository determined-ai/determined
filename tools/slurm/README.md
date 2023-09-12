# Starting up a slurmcluster on GCP

## Quick start

1. Install Terraform following [these instructions](https://developer.hashicorp.com/terraform/downloads).
2. Download the [GCP CLI](https://cloud.google.com/sdk/docs/install-sdk) and run `gcloud auth application-default login` to get credentials.
3. Run `make slurmcluster` from the root of the repo and wait (up to 10 minutes) for it to start.
   - To specify which container runtime environment to use, pass in `FLAGS="-c {container_run_type}"` to `make slurmcluster`. Choose from either `singularity` (default), `podman`, or `enroot`.
   - To specify which workload manager to use, pass in `FLAGS="-w {workload_manager}"` to `make slurmcluster`. Choose from either `slurm` (default) or `pbs`. Note: in specifying the workload manager, `make slurmcluster` will automatically load the appropriate boot disk image (found in `terraform/images.conf`).
   - The default configuration yields a Slurm cluster with a single compute node and 8 CPUs (`n1-standard-8`).   You can control the machine_type, and gpus of the compute node using `FLAGS="-m {machine_type} -g {gpu_type}:{count}"`.  See below.
   - By default, all VMs created with `make slurmcluster` will be destroyed after 7200 seconds (2 hours). To specify a different amount of time, pass in `FLAGS="-t {time_seconds}"` to `make slurmcluster`.
4. Step 2 will ultimately launch a local devcluster. Use this as you typically would [1].
5. Release the resources with `make unslurmcluster` when you are done.

Once the VM is created and SSH connection is established, one can also directly connect to the created instance with the command 
```
gcloud compute ssh --zone <zone_name> <instance_name> --project <project_name>
```

For example:
```
gcloud compute ssh --zone us-west1-b phillipgaisford-dev-box --project determined-ai
```

[1] It is fine to exit this and restart it with `make slurmcluster` again as you please.

To see usage of the `make slurmcluster` target, run `make -C tools/slurm usage`.

## GPU Configuration

The configuration of GPUs on GCP requires matching specific `machine_type`, gpu type, and gpu count combinations.
See the [GCP GPU platforms](https://cloud.google.com/compute/docs/gpus) documentation page for specifics.
Additionally, the `NodeName` definition in `/etc/slurm/slurm.conf` must be configured to properly reflect the number and type of gpus available
for them to be visible in Slurm.   This is currently handled by the custom `terraform/scripts/startup-script.sh` which dynamically discovers the GPU count and
type for the local dev-box using `nvidia-smi` and injects the Gres attribute for all `NodeName` definitions in `slurm.conf`.

The default `machine_type` is `n1-standard-8` and can support 1, 2 or 4 gpus of type `nvidia-tesla-t4`, `nvidia-tesla-p4`, `nvidia-tesla-v100`, or `nvidia-tesla-p100`.

Some example GPU configuration that work are listed below.   Optionally combine with `-c {container_run_type}` and `-w {workload_manager}` for your desired testing configuration.

### Example GPU Configuration recipes for `n1-standard-8`
 - `FLAGS="-g nvidia-tesla-t4:2"`
 - `FLAGS="-g nvidia-tesla-t4:4"`
 - `FLAGS="-g nvidia-tesla-p4:4"`
 - `FLAGS="-g nvidia-tesla-v100:2"`
 - `FLAGS="-g nvidia-tesla-v100:2"`

### Example GPU Configuration recipes for `g2-standard-8`
 - `FLAGS="-m g2-standard-8 -g nvidia-l4:1"`
 - `FLAGS="-m g2-standard-48 -g nvidia-l4:4"`

Other GPU types require that you select a proper `machine_type` and gpu type and count as per [GCP GPU platforms](https://cloud.google.com/compute/docs/gpus).
GPUs are charged per-hour.  `nvidia-tesla-4` model are the cheapest to use.



## Alternatives

The `make slurmcluster` flow is fast and convenient. Alternatively, if you use
`make -C terraform build` and then just use the resulting instance as a dev box after 
installing Determined and VS Code Remote on it, the experience is better after getting it
setup (though this somewhat is a matter of preference).

## `make slurmcluster` notes

To run Determined + Slurm, you have a few options:

- Spin up a Linux development machine, install all the prerequisite software and run a cluster as a
  customer would, following our publicly available documentation.
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

By default, each devbox invoked by `make slurmcluster` will automatically delete the VM after two hours of runtime. If you want to override this time limit, one can run `FLAGS="-t {time_seconds}"`. Where `seconds` is a value between 0 to 10,281,600 seconds inclusive. The two hour time limit ensures that devboxes are being deleted if they are not used to prevent excess costs.

## Running a Slurmcluster on a Developer Launcher

Despite each VM being intended for a single user, per-user deployment of the Launcher is still useful for quickly testing changes to Launcher (instead of building a new image with the updated Launcher which can take up to 40 minutes). 

One can load a developer launcher on their dev box created by `make slurmcluster` with this workflow:
1. From the root of this repository, run `make slurmcluster` to create a VM. **Note**: this will also start a devcluster pointing at port 8081 on the VM *automatically* (which is not the desired Launcher port in this case). **After the VM is created, terminate the `make slurmcluster` process.**
2. Obtain the external IP address of the created VM by running  

    ```
    gcloud compute ssh --zone [ZONE_ID] [VM_NAME] --project [PROJECT_ID] -- "curl https://ifconfig.me/"
    ```

    For example:

    ```
    gcloud compute ssh --zone "us-west1-b" quilici-dev-box --project "determined-ai" -- "curl https://ifconfig.me/"
    ```
3. In the [hpc-ard-capsules-core repository]([d.com](https://github.hpe.com/hpe/hpc-ard-capsules-core)), run `./loadDevLauncher.sh -g [$USER]@[EXTERNAL_IP]` which will spin up a developer launcher on port 18080 on the specified VM.
4. From the root of this repository, run `make slurmcluster FLAGS="-d"` which will start a devcluster pointing at port 18080 on the instance.

## Using Slurmcluster with Determined Agents

`make slurmcluster` supports using Determined agents to run jobs. To do this with `make slurmcluster` do the following steps from the `determined-ee/` directory:

1. `make -C agent build package`
2. `make slurmcluster FLAGS="-A"`
3. `gcloud compute scp agent/dist/determined-agent_linux_amd64_v1/determined-agent $USER-dev-box:/home/$USER --zone us-west1-b`

The `FLAGS="-A"` in `make slurmcluster` removes the resource_manager section in the slurmcluster.yaml that would otherwise be used. This then defaults to the agent rm and the master waits for agents to connect and provide resources. The scp command brings the determined-agent to the dev-box. `$USER` will be replaced with your username when initiating GCP.

Then, connect to your dev-box. This can be done with `make -C tools/slurm/terraform connect` or `gcloud compute ssh $USER-dev-box --project=determined-ai --zone=us-west1-b`. Input the following command on the devbox in order to allocate resources on slurm.

`srun $HOME/determined-agent --master-host=localhost  --master-port=8080 --resource-pool=default --container-runtime=singularity`

You can also use podman by changing the value for `container-runtime` to `podman`. 

This command allocates the 8-core CPU that is on the GCP machine. Unfortunately, there are currently no gpus on the VM so we can not allocate any. 

Now, you can launch jobs like normal using the Determined CLI. You can check the status of the allocated resources using `det slot list`.

If you encounter an issue with jobs failing due to `ModuleNotFoundError: No module named 'determined'` run `make clean all` to rebuild determined. 
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
-k 'not cifar10_pytorch_distributed'
```

This invocation specifies that all tests are to be run via the remote launcher running on `localhost:8080`. The command specifies that all pytests with the `e2e_slurm` mark should be run except those that have the `parallel` mark as well (this is due to the fact that there are no GPUs and only 1 node on the GCP compute instances). The `and not cifar10_pytorch_distributed` specifies also not to run the `cifar10_pytorch_distributed` test. This test is omitted because it takes too long to run on instances with only one node.

# Notes on `make slurmcluster` tests on CircleCI 

By default, the `test-e2e-*-gcp` jobs are not run within the `test-e2e` workflow on a **developer branch**. If you would like to invoke these jobs on a certain commit, you must add the `ci-run-allgcp` label to your pull request on github.

**On branch `main` and `release/rc` branches, these jobs always run without needing to set the `ci-run-allgcp` label.**

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

# Base Image Updates
The `make slurmcluster` tool uses a base image `det-environments-slurm-ci-###` or `det-environments-pbs-ci-###` for Slurm or PBS respectively.   These images are referenced by
the configuration file `tools/slurm/terraform/images.conf`.  The images contain:

- Pre-configured Slurm or PBS
- Nvidia drivers
- The HPC launcher
- A singularity cache loaded with the current default CPU and CUDA task environments at the time the image was created
- A enroot sqsh file cache `/svc/enroot` with the current default CPU and CUDA task environments at the time the image was created

These images must be updated periodically with a new launcher, and task environments.   See [packer](packer/README.md)

