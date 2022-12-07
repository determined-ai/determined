# Experiment Config Development Guide

- We use [JSON Schema](https://json-schema.org/) to define the schema of
  the experiment configuration. This helps apply cross-language policies
  for validation, null handling, default values, and some custom rules.

- To make changes to the configuration schema, change the according files below:
  - See `schemas/expconf` for logics that are shared across languages.
  - See `schemas/test_cases` for test cases that are shared across languages.
  - See `master/pkg/schemas/expconf` for go struct definitions.
  - See `harness/determined/common/schemas/expconf` for python class definitions.

- We generate code that contains the definitions and utility functions of
  structs.  See `schemas/gen.py`.

- Validation:
  - Validation happens in two steps:
    - Sanity check: make sure everything is parsable
    - Validation: make sure everything is present
  - Sanity check is applied to raw user inputs
  - Validation is applied after cluster defaults have been applied
  - The implementation difference between sanity check and validation is the
    inclusion of a single `eventuallyRequired` schema in the latter.

- Null handling:
  - Anything with a null value is treated as not present.
  - Reason: there is no pythonic or golangic way to represent values which were
    provided in the configuration as literal nulls, rather than values which
    were not provided at all.  In theory, you could have singleton
    "NotProvided" pointers, which you would check for every time that you
    checked a value in the schema, but in practice that would be a pain.
    Additionally, the golang json-schema library we use treats not-present
    values a nil values anyway.

- Default values:
  - Default values are defined in JSON Schema.
  - However, other default values might be populated on the master side.

- Customization rules:
  - We define some JSON schema extensions to provide the behavior we need.
    JSON Schema itself does not provide the behavior we want out of the box.
    Most JSON Schema libraries have great support for custom validation logic.
  - We have json-schema extensions for the following keywords:
    - `checks`: Custom messages for arbitrary validation logic
    - `compareProperties`: Support customizable value comparisons
    - `disallowProperties`: Custom messages when disallowing properties
    - `eventuallyRequired`: Support two-step validation
    - `union`: Excellent error messages when validating union types
    - `optionalRef`: like `$ref`, but only enforced for non-null values
    - `eventually`: Defer validation of inner clause till completeness validation phase
  - The canonical implementations (with thorough comments) may be found in
    `harness/determined/common/schemas/extensions.py`.

- Migration and Versioning:
  - Migration logics are implemented in the master. See `pkg/schemas/expconf/parse.go`.
  - Bumping the version is only necessary when previously-valid configs become
    no longer valid.  For instance, removing a once-available field requires a
    version bump, but appending a new field or accepting a new type in an
    existing field does not.
  - To bump versions, take the following steps (assume you are adding `v2`):
    - create a new `v2` directory, like `schemas/expconf/v2`
    - copy the json schemas files you are updating from the `v1` directory
      into the `v2` directory.  Also copy any files which referenced any of
      those files; for example, if you update `v1/hyperparameters.json`,
      you'll have to copy `v1/experiment-config.json` as well since its
      `$ref` will need to now point to the `v2/hyperparameters.json`.
    - Populate `test_cases/v2` with test cases to the new json schemas you
      just made.
    - Make the golang changes:
      - Create the corresponding objects in `pkg/schemas/expconf`, such as a
        new `ExperimentConfigV2` and new `HyperparametersV2`
      - update `expconf/latest.go` to point to the correct new object
      - update any uses of that object throughout the codebase to reflect
        the new structure.
      - ensure that `make gen && go test ./pkg/schemas/expconf` passes
    - Currently, no python or JS changes are needed.
