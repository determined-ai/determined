# tools/scripts

This directory is the home for tools that directly assist in the development of
the determined repository.

## `bumpenvs`: How to bump task environment versions

Our task environments are versioned separately from the main determined
repository. The motivation for this is that customers who maintain custom
Docker images as extensions to our image should not have to rebuild their
custom images as often as we cut releases. Since in practice our prebuilt
environments are pretty slow to change, and since some organizations have a
long and tedious security review for task environments, this is a good thing
for customers. But it does mean that updates to the environments repo have to
be registered in the determined repo. Here is the process:

1. Land the desired change in the environments repo. Remember the full commit
   hash (we'll call it `THECOMMIT` in these steps).

2. Wait for the post-merge-to-master CircleCI jobs on the environments repo to
   finish. These will publish the relevant Docker/AWS/GCP images and create
   machine-readable artifacts containing the image tags.

3. Enter the `tools/scripts` directory of the determined repo.

4. Run `./update-bumpenvs-yaml.py bumpenvs.yaml THECOMMIT`. This will fetch the
   above-mentioned machine-readable artifacts from the CircleCI jobs of the
   environments repository, parse out the image tags, and update the relevant
   entries in `bumpenvs.yaml`.  For every artifact found, this will set that
   artifact's `old` value to the previous `new` value, and set the new `new`
   value to the artifact produced by CI, including the task environments and
   the agent AMIs.

5. (optional) Run `./refresh-ubuntu-amis.py bumpenvs.yaml`.  This will fetch
   the up-to-date Ubuntu AMIs for each region for each of the `*_master_ami`
   and `*_bastion_ami` image tags in bumpenvs.yaml.  This isn't strictly
   necessary; we just need to run it periodically, and now is a fine time.

6. Run `./bumpenvs.py bumpenvs.yaml`.  This will do a simple string replacement
   in the repository, replacing the `old` values with the `new` values for
   every image type identified in `bumpenvs.yaml`.
