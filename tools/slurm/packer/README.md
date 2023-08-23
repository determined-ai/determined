# Building Images with Packer

This sub-repository builds the image `make slurmcluster` uses. The Makefile is the best
documentation for interacting with and building this code. To build, you will need to install
`packer`  (`brew install packer` and run `packer init` on the packer directory as below) 
and have `gcloud` (see [Slurmcluster README.md](../README.md)), then run `make build`. 

It was last built with `Packer v1.8.6`.

## Packer Initialization

This needs to be done prior to `packer build`:

`packer init tools/slurm/packer`

## How to Build an Image

After the pre-requisite software is installed, one can run `make build WORKLOAD_MANAGER=[type]` where `type` is either `slurm` or `pbs` (default value is `slurm`) to build a SLURM or PBS image, respectively. Upon successful completion of the build, the image name will be placed in the appropriate value in `../terraform/images.conf` (either `slurm` or `pbs` depending on what was specified).

This process builds images named `det-environments-slurm-ci-###` or `det-environments-pbs-ci-###` and have the image family set to
[det-environments-slurm-ci](https://console.cloud.google.com/compute/images?tab=images&authuser=0&project=determined-ai&pageState=(%22images%22:(%22p%22:0,%22r%22:200,%22f%22:%22%255B%257B_22k_22_3A_22Family_22_2C_22t_22_3A10_2C_22v_22_3A_22_5C_22det-environments-slurm-ci_5C_22_22_2C_22i_22_3A_22family_22%257D%255D%22,%22s%22:%5B(%22i%22:%22creationTimestamp%22,%22s%22:%221%22),(%22i%22:%22type%22,%22s%22:%220%22),(%22i%22:%22name%22,%22s%22:%220%22)%5D)))

These images need to be periodically pruned.   The launcher version included in the image can be seen in the description field.


## 'Publishing' Updated Images

When building a new image for `make slurmcluster WORKLOAD_MANAGER=[type]` the build will use the `hpe-hpc-launcher-*.deb` debian located in `tools/slurm/packer/build`. If there is none present, a script will download and build with the latest launcher version. The value for the generated image (either SLURM or PBS) in `../terraform/images.conf` is automatically updated with the newly built image after the build finishes (depending on the workload manager specified). The workflow for building and updating the image with the latest released launcher should be as follows:

1. Checkout clean branch
2. `make -C tools/slurm/packer clean build`  
3. `make -C tools/slurm/packer build WORKLOAD_MANAGER=pbs`
4. `git add  tools/slurm/terraform/images.conf`
5. `git commit`
6. Post PR to update the default images.
7. Manually prune the images [det-environments-slurm-ci](https://console.cloud.google.com/compute/images?tab=images&authuser=0&project=determined-ai&pageState=(%22images%22:(%22p%22:0,%22r%22:200,%22f%22:%22%255B%257B_22k_22_3A_22Family_22_2C_22t_22_3A10_2C_22v_22_3A_22_5C_22det-environments-slurm-ci_5C_22_22_2C_22i_22_3A_22family_22%257D%255D%22,%22s%22:%5B(%22i%22:%22creationTimestamp%22,%22s%22:%221%22),(%22i%22:%22type%22,%22s%22:%220%22),(%22i%22:%22name%22,%22s%22:%220%22)%5D))) retaining the most recent 6 images or so.

To build with a specific launcher version, put the `hpe-hpc-launcher-*.deb` in the `tools/slurm/packer/build` directory and run `make -C tools/slurm/packer build` (without the `clean` option).

Make sure you are on the vpn or have credentials to access arti.hpc.amslabs.hpecorp.net to download the latest launcher version.

`make slurmcluster` is pinned to a specific image, not the image family, so just building will
not cause (potentially destructive) updates to anyone using it. If you do publish the change
by committing it and someone picks up your change, by default, `make slurmcluster` does not
`--auto-approve` its Terraform plans so others will get a warning if it affects them.

# When to do this

This should also be done as part of the standard release process for each new HPC Launcher version published so that we are always testing with the latest released HPC launcher. Running `packer build` uses the `scripts/generate-pkr-vars.sh` script to automatically detect if the local Launcher version is out of date and prompts the developer if they would like to replace the local outdated version with the newest version.   Always publish both Slurm & PBS versions.
