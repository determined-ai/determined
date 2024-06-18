<!---
## PR TITLE (Commit Body)
When squash-merging, GitHub will use this as the commit message.
Check the "Example Commit Body" for conventional commit semantics.
-->
## Ticket
<!---
A reference to the Jira ticket or Github issue. e.g. "[DET-1234]" or #123.
-->



## Description
<!---
A description of the PR. For breaking changes, lead with "BREAKING CHANGE:".
--->



## Test Plan
<!---
Describe the scenarios in which you've tested your change, with screenshots as
appropriate. Reviewers may ask questions about this test plan to ensure adequate
coverage of changes.
-->



## Checklist

- [ ] Changes have been manually QA'd
- [ ] New features have been approved by the corresponding PM
- [ ] User-facing API changes have the "User-facing API Change" label
- [ ] Release notes have been added as a separate file under `docs/release-notes/`
  See [Release Note](https://github.com/determined-ai/determined/blob/master/docs/release-notes/README.md) for details.
- [ ] Licenses have been included for new code which was copied and/or modified from any external code

<!---
Example Commit Body:
docs: tweak recommended "pip install" usage [DET-123]

Specifically, this title should contain a type and a description
of the change being made:

User-facing change types:

- docs: docs-only change
- feat: new user-facing feature
- fix: bug fix
- perf: performance improvement

Internal change types:

- build: build system change (anything in a `Makefile`, mostly)
- chore: any internal change not covered by another type
- ci: anything that touches `.circleci`
- refactor: internal refactor
- style: style change
- test: new tests

See https://www.conventionalcommits.org/en/v1.0.0/ for background.

The first line should also:

- be at most 89 characters long
- contain a description that is at most 72 characters long
- not end with sentence-ending punctuation
- start (after the type) with a lowercase imperative ("add", "fix")
-->
