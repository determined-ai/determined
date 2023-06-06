## Quick start

1. Install Terraform following [these instructions](https://developer.hashicorp.com/terraform/downloads).
2. Download the [GCP CLI](https://cloud.google.com/sdk/docs/install-sdk) and run `gcloud auth application-default login` to get credentials.
3. Run `make slurmcluster` from the root of the repo and wait (up to 10 minutes) for it to start.
4. Step 2 will ultimately launch a local devcluster. Use this as you typically would [1].
5. Release the resources with `make unslurmcluster` when you are done.

[1] It is fine to exit this and restart it with `make slurmcluster` again as you please.

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

Under the hood, this launches a compute instance with Slurm, Singularity (Apptainer), the Cray
Launcher component and many other dependencies pre-installed. Then, SSH tunnels are opened so that
`localhost:8081` on your machine points at `localhost:8081` on compute instance and
`localhost:8080` on the compute instance points at `localhost:8080` on your machine. Last,
`devcluster` is started with the Slurm RM pointed at the remote instance, and local development
with `devcluster` works from here as always.
