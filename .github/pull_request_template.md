## Description

<!---
Lead with the intended commit body in this description field. For breaking changes,
please include "BREAKING CHANGE:" at the beginning of your commit body.
At a minimum, this section should include a bracketed reference to the Jira ticket,
e.g. "[DET-1234]". When squash-and-merging, copy this directly into the description field.
-->


## Test Plan

<!---
Describe the situations in which you've tested your change, and/or a screenshot as appropriate.
Reviewers may ask questions about this test plan to ensure adequate manual coverage of changes.
-->


## Commentary (optional)

<!---
Use this section of your description to add context to the PR. Could be for particularly
tricky bits of code that could use extra scrutiny, historical context useful for reviewers, etc.
You may intentionally leave this section blank and remove the title.
--->


## Checklist
- [ ] Changes have been manually QA'd
- [ ] User-facing API changes need the "User-facing API Change" label.
- [ ] Release notes should be added as a separate file under `docs/release-notes/`.
See [Release Note](https://github.com/determined-ai/determined/blob/master/docs/release-notes/README.md) for details.
- [ ] Licenses should be included for new code which was copied and/or modified from any external code.
- [ ] If modifying `/webui/react/src/shared/` verify `make -C webui/react test-shared` passes.


<!---
## Title

Example title: "docs: tweak recommended "pip install" usage".

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
