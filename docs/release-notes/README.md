# Release Notes

This process is intended to ensure accurate and comprehensive release notes,
so Determined users can understand what has changed and the impact it has on
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

### Procedure

1. Create a release note as part of landing the change, itself, in the
   same PR or as part of a chain of PRs of a large feature.

   The author of the PR has the most context about the change being made and should be
   aware of any caveats or additional context that users should be informed about.

1. Write the release note entry as a separate file and add it to the
   `determined/docs/release-notes` directory. See the [example](#release-note-example), below.

   Writing guidelines can be relaxed for a release note to be more
   conversational than might be acceptable in the core documentation. Spelling,
   grammar, coherence, and completeness requirements still apply. Passing
   [Grammarly](https://app.grammarly.com/) checks is usually sufficient.

   * Give the file a descriptive name related to the change.
   * Write the release note using [reStructuredText](https://www.sphinx-doc.org/en/master/usage/restructuredtext/index.html), following Determined documentation [style conventions](https://determinedai.atlassian.net/l/c/53h3PrPo).
   * Include the `:orphan:` [metadata directive](https://www.sphinx-doc.org/en/master/usage/restructuredtext/field-lists.html#metadata) to suppress warnings about the file not being included in the table of contents. Later, the file is merged into the final release note and deleted.
   * Specify one or more of the following categories, depending on the scope of the change:

     * `**Bug Fixes**`
     * `**Security Fixes**`
     * `**Breaking Changes**`
     * `**Improvements**`
     * `**New Features**`

   * Enter a title for the change, or titles for each change, as one or more list elements.

     Begin each title with a prefix for the applicable component:

     * `WebUI`
     * `Notebook`
     * `TensorBoard`
     * `Command`
     * `Shell`
     * `Experiment`
     * `API`
     * `Images`
     * other component.

   * Provide a short, descriptive, summary title.

     **Note:** For a release note that might have particular significance for the user, use an `Important` admonition and highlight it. For example:

     > **Security Fixes**
     >
     > *  CLI: **Important:** API requests executed through the Python bindings have been erroneously using the SSL
     >    "noverify" option since version 0.17.6, making them potentially insecure. The option is now disabled.

   * In one or more list items, provide additional information. Describe:

     * what changed
     * why the change was needed
     * how the change affects the user

       Do not give details about how the change was implemented.

       If there is a Jira issue associated with the change, the Jira **Description** field, with some rewording can be used as the description.

       Be sure to highlight API changes and backward incompatibility, including steps needed to upgrade safely.

       Do not include:

       * links to Determined documentation
       * customer identifiers
       * internal project status or plans
       * Jira issue or PR identifiers


   * Use double backticks, not single backticks, for inline code samples.

     * Incorrect

       ```markdown
       The `foo(bar)` endpoint no longer accepts `waldo` and now accepts a `garply`.
       ```

     * Correct

       ```markdown
       The ``foo(bar)`` endpoint no longer accepts ``waldo`` and now accepts ``garply``.
       ```

   * It is better to spell out acronyms or abbreviations. For example:

     * Incorrect

       k8s: Fix a crash affecting namespaces.

     * Correct

       Kubernetes: Fix a crash affecting namespaces.



### Release Note Example

```markdown
:orphan:

**New Features**

*  GCP: Add support for provisioning Nvidia A100 GPU instances.

   *  Running workloads on A100 chips currently requires building a custom task
      environment with CUDA 11, because the default task environments provided
      by Determined contain either CUDA 10.0 or CUDA 10.1. The default task
      environments will be upgraded to CUDA 11 in a future release of
      Determined.
```

## How to Collect and Publish the Release Notes for a Release

As part of the release process, the release manager:

1. Prepends the individual `docs/release-notes/` files to the `docs/release-notes.txt` file and creates a new version heading.
1. Deletes the individual files from `docs/release-notes/`.
1. Performs additional copy editing as needed.
