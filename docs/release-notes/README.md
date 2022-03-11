# Release Notes

Release notes are to be accurate and comprehensive, with the objective that
Determined users can understand what has changed and the impact it has on
their use of the product.

The work required to write high-quality release notes is intended to be
spread over time and across project contributors instead of writing all of
the release notes at the last minute as part of cutting a new release.

## How To Write an Individual Release Note

Create a release note when you make a change that should be communicated to users,
classifying the release note according to one of the following categories:

* Bug Fixes
* Breaking Changes
* Improvements
* New Features

Be sure to   highlight API changes and backward incompatibilities, discuss any steps
that must be taken to upgrade safely.

Writing guidelines can be relaxed for a release note to be more conversational than might
be acceptable in the core documentation. Spelling, grammar, coherence, and completeness
requirements still apply. Passing [Grammarly](https://app.grammarly.com/) checks is usually
sufficient.

Generally, you should not link to other locations in the documentation. Release notes have
a long life across versions of the system and it is highly likely the link will eventually
break.

### Procedure

#. Create a release note as part of landing the change, itself, in the
   same PR or as part of a chain of PRs of a large feature.

     The author of the PR has the most context about the change being made and should be
     aware of any caveats or additional context that users should be informed about.

#. Write the release note entry as a separate file and add it to the
   `determined/docs/release-notes` directory. See the example, below.

    * Name the file with the PR number as a prefix. For example, `1097-nvidia-a100-support.txt`.
    * The file should be in [reStructuredText](https://determinedai.atlassian.net/l/c/53h3PrPo) format
      and should start with `:orphan:` metadata string to avoid errors when building the docs.
    * Specify one or more of the following categories, depending on the extent of the change:

        * Bug Fixes
        * Breaking Changes
        * Improvements
        * New Features

    * Enter a title for the change, or titles for each change, as one or more list elements.

        Begin each title with a prefix for the affected component:
        
          * WebUI
          * Notebook
          * TensorBoard
          * Command
          * Shell
          * Experiment
          * API
          * Images
          * or other applicable component.

    * Provide a short, descriptive, summary title.
    * In one or more list elements, provide more detailed description(s) of the change.

        Describe what changed, why the change was needed, and how the
        change affects the user. Do not give details of how the change was implemented.
        If there is a Jira issue associated with the change, the Jira description can be
        helpful as a guide to what context should also be provided in the release note.

        Do not include:

        * links to Determined documentation
        * customer identifiers
        * internal project status or plans
        * Jira issue or PR identifiers

#. As part of the PR review, a documentation team member is assigned as a reviewer and
   reviews and approves the release note part of the PR.

### Release Note Example

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

## How to Create the Release Notes for a Release

* As part of the release process, the release manager will merge these files
  together into `docs/release-notes.txt`, delete the individual files from
  `docs/release-notes/`, and then do additional copy-editing and formatting as
  necessary.
