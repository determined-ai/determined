# Release Notes

We want release notes that are accurate, comprehensive, and written so that
users of Determined can understand what has changed and what impact it has on
their use of the product. We also want to spread out the work required to write
high-quality release notes over time and across the contributors to the project,
rather than writing all of the release notes at the last minute as part of
cutting a new release.

## How To Write Release Notes

* Include an update to the release notes whenever you make a change that should
  be communicated to users. That includes bug fixes, new features, changes to
  previous behavior/APIs, and improvements to existing features. Be sure to
  highlight API changes and backward incompatibilities, discuss any steps that
  must be taken to upgrade safely, and link to other locations in the documentation
  as needed.

* Update the release notes as part of landing the change itself (e.g., in the
  same PR or as part of a chain of PRs to land a large feature). The author of
  the PR has the most context about the change being made and should be aware of
  any caveats or additional context that users should be informed about.
  Reviewers should also look at proposed release note changes as part of
  reviewing a PR.

* Write the release note entry as a separate file and add it to the
  `docs/release-notes` directory.

  * Name the file with the PR number as a prefix. For example, `1097-nvidia-a100-support.txt`.
  * The file should be in reStructuredText format and should start with
    `:orphan:` to avoid errors when building the docs.
  * The file should contain one or more list elements, which should be
    categorized into one of **Breaking Changes**, **New Features**, 
    **Improvements**, **Bug Fixes**.
  * Each item needs to have a prefix of which component it is about.
    The prefix needs to be one of **WebUI**, **Notebook**, **TensorBoard**,
    **Command**, **Shell**, **Experiment**, **API**, **Images**, or any
    applicable component.
  * For example:

    ```
    :orphan:

    **New Features**

    - GCP: Add support for provisioning Nvidia A100 GPU instances.

      - Running workloads on A100 chips currently requires building a custom task
        environment with CUDA 11, because the default task environments provided by
        Determined contain either CUDA 10.0 or CUDA 10.1. Refer to the
        :ref:`custom-env` documentation for more details. The default task
        environments will be upgraded to CUDA 11 in a future release of Determined.
    ```

* As part of the release process, the release manager will merge these files
  together into `docs/release-notes.txt`, delete the individual files from
  `docs/release-notes/`, and then do additional copy-editing and formatting as
  necessary.
