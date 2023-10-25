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