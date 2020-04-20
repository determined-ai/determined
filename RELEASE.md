# Releases

This document describes the process for cutting and publishing a new version
of Determined. Please read this document carefully before proceeding with a release.

## Prerequisites

- Terraform 0.12+
- For prerequisites to build cloud images, see [build cloud images](cloud/README.md#Prerequisites) for details.
- Push/pull access to determined-ai/determined-examples

### Installing Terraform

For Mac OS X, Terraform is available on homebrew:

```bash
brew install terraform
```

For Linux, Terraform is available through a direct download:

```bash
sudo apt-get install unzip wget
wget https://releases.hashicorp.com/terraform/0.12.9/terraform_0.12.9_linux_amd64.zip
unzip terraform_0.12.9_linux_amd64.zip
sudo mv terraform /usr/local/bin/
```

### Configuring AWS Credentials

Two methods are available for supplying AWS credentials: environment variables and a credentials file.

For environment variables:

```bash
export AWS_ACCESS_KEY_ID="anaccesskey"
export AWS_SECRET_ACCESS_KEY="asecretkey"
```

For credential files:

```bash
mkdir -p $HOME/.aws/
cat > $HOME/.aws/credentials << EOF
[default]
aws_access_key_id = anaccesskey
aws_secret_access_key = asecretkey
EOF
```

See [AWS Authentication](https://www.terraform.io/docs/providers/aws/index.html#authentication) for more help.

## Cutting the Release

1. Switch to the master branch:

```bash
git checkout master
```

2. Ensure that the master branch is in a good state (e.g., passes CI).

3. Ensure the release notes cover all significant changes in this release, and update to the correct release date.

4. Commit and push the updated release notes to the main Determined repo.

```bash
git add <RELEASE-NOTES>
git commit -m "Update release notes."
```

5. Tag the release and push the tag to the main Determined repo:

```bash
git tag v0.12.2
git push upstream v0.12.2
```

6. Build Determined, publish the tarball, and publish the Determined images (cloud and docker):

```bash
make publish
```

**NOTE:** This assumes you have logged in to an authorized Docker Hub
account via `docker login`.

7. Update the version number to the next working version (this automatically commits the change but does not push it). This defaults to a patch semantic version update (e.g. 0.4.9->0.4.10). If you want to do a minor or major version update, set the PART environment variable to `minor` or `major` respectively.

```bash
make bump-version
```

8. Push the changes.

```bash
git push
```
