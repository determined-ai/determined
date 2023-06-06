## Building the image

This sub-repository builds the image `make slurmcluster` uses. The Makefile is the best
documentation for interacting with and building this code. To build, you will need to install
`packer` and have `gcloud`, then run `make build`. 

It was last built with `Packer v1.8.6`.

## 'Publishing' the udpated image

To update the image `make slurmcluster` uses, after building, change the default value for
`vars.boot_disk` in `../terraform/variables.tf` to the new image and commit the change.

`make slurmcluster` is pinned to a specific image, not the image family, so just building will
not cause (potentially destructive) updates to anyone using it. If you do publish the change
by committing it and someone picks up your change, by default, `make slurmcluster` does not
`--auto-approve` its Terraform plans so others will get a warning if it affects them.

# When to do this

Whenever it breaks, or you want to add something.
