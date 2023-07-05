# Building Images with Packer

This sub-repository builds the image `make slurmcluster` uses. The Makefile is the best
documentation for interacting with and building this code. To build, you will need to install
`packer` and have `gcloud`, then run `make build`. 

It was last built with `Packer v1.8.6`.

## How to Build an Image

After the pre-requisite software is installed, one can run `make build WORKLOAD_MANAGER=[type]` where `type` is either `slurm` or `pbs` (default value is `slurm`) to build a SLURM or PBS image, respectively. Upon succesful completion of the build, the image name will be placed in the appropriate value in `../terraform/images.conf` (either `slurm` or `pbs` depending on what was specified).

## 'Publishing' the updated image

When building a new image for `make slurmcluster WORKLOAD_MANAGER=[type]` the build will use the `hpe-hpc-launcher-*.deb` debian located in `tools/slurm/packer/build`. If there is none present, a script will download and build with the latest launcher version. The value for the generated image (either SLURM or PBS) in `../terraform/images.conf` is automatically updated with the newly built image after the build finishes (depending on the workload manager specified). The workflow for building and updating the image with the latest released launcher should be as follows:

1. Checkout clean branch
2. make -C tools/slurm/packer clean build
3. git add  tools/slurm/terraform/images.conf
4. git commit
5. Post PR to update the default image.

To build with a specific launcher version, put the hpe-hpc-launcher-*.deb in the `tools/slurm/packer/build` directory and run `make -C tools/slurm/packer build`.

Make sure you are on a vpn or have credentials to access arti.hpc.amslabs.hpecorp.net to download the latest launcher version.

`make slurmcluster` is pinned to a specific image, not the image family, so just building will
not cause (potentially destructive) updates to anyone using it. If you do publish the change
by committing it and someone picks up your change, by default, `make slurmcluster` does not
`--auto-approve` its Terraform plans so others will get a warning if it affects them.

# When to do this

Whenever it breaks, or you want to add something.
