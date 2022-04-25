# Shared UI

We this directory for shareable code snippets, UI components, etc.

- separate reusable and shareable code across different Determined UIs
- leave build and transpliation to target projects
- stepping stone for going to a solution such as packaging the components individually
or separately using Bit or Node registry.
- assume mostly similar build and dev environments:
  - lower overhead, faster start up, simpler, faster, and smaller builds
  - familiarity: no new code management tool to learn

Limitations:
- shared dependencies between all the projects would need to have similar APIs
- no pre-canned component-level versioning solution

## How

TODO instructions for sharing new components, updating existing ones, deprecating, etc.

Code in this directory should not depend on internal dependencies that leave outside this directory
