## Building the image

This sub-repository builds the image `make slurmcluster` uses. The Makefile is the best
documentation for interacting with and building this code. To build, you will need to install
`packer` and have `gcloud`, then run `make build`. 

It was last built with `Packer v1.8.6`.

## 'Publishing' the updated image

When building a new image for `make slurmcluster` the build will use the `hpe-hpc-launcher-*.deb` debian located in `tools/slurm/packer/build`. If there is none present, a script will download and build with the latest launcher version. The value for `vars.boot_disk` in `../terraform/variables.tf` is automatically updated with the newly built image after the build finishes. The workflow for building and updating the image with the latest released launcher should be as follows:

1. Checkout clean branch
2. make -C tools/slurm/packer clean build
3. git add  tools/slurm/terraform/variables.tf
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
