# Execution flow

We execute each CircleCI pipeline in the following order:
* `config.yml` (as required by CircleCI itself)
* `real_config.yml` (called from `config.yml`)

## config.yml

This config defines a single workflow. Its task is to set parameters for the actual CI worksflows that are defined in `real_config.yml`.

Its order of execution:
1. (job `set_up_param_file`) Assemble configuration from various sources into a single JSON file
2. (job `set_up_param_file`) Persist that JSON file to a workspace (shared by the entire workflow) so other jobs within the workflow can access it
3. (job `exec_config`) Attach the shared workspace to a file path
4. (job `exec_config`) Execute `real_config.yml`, passing in parameters contained in the JSON from the workspace

As of this writing, `config.yml` sets the following parameters:
* `do_nightly_tests`: a boolean that nightly (and nightly-quarantine) tests can use to determine whether they should run. This will be true if either the pipeline is executed from the nightly trigger or the PR contains the label `ci-run-nightly`.

## real_config.yml

This is the file where all tests are currently defined. New steps, jobs, and workflows related to executing actual tests should be added here.

# Docker images for CircleCI (package-and-push-system)

The Package and Push System builds and distributes Docker images for master and agent. There are two primary jobs in this system that handle Docker images differently depending on where and how they will be used:

## package-and-push-system-local
This job builds Docker images for the master and agent and saves them locally on the CircleCI executor. Images built by this job are not pushed to Docker Hub or any remote image repository. They're only available on the CircleCI machine that performed the build.

## package-and-push-system-dev
This job builds Docker images and pushes them to Docker Hub. After this step is run, the images are globally available for different deployment environments (like `det deploy` on AWS or Kubernetes). This step is required to be upstream of any tests or deployments that require Docker images to be accessible from external environments.

In cases where a Docker image is needed by job "Downstream", and `package-and-push-system-dev` is omitted from Downstream's `requires` section, Downstream might still succeed if another job has previously pushed the relevant Docker image for the same Git commit. This is true even if the job that ran `package-and-push-system-dev` is in a different workflow or pipeline.