import os
from typing import Any, Dict, Iterator, List

import jsonschema


def disallowProperties(
    validator: jsonschema.Draft7Validator, disallowed: Dict, instance: Any, schema: Dict
) -> Iterator[jsonschema.ValidationError]:
    """
    disallowProperties is for restricting which properties are allowed in an object with
    per-property, such as when we allow a k8s pod spec with some fields disallowed.

    Example: The "pod_spec" property of the environment config:

        "pod_spec": {
            "type": "object",
            "disallowProperties": {
                "name": "pod Name is not a configurable option",
                "name_space": "pod NameSpace is not a configurable option"
            }
        }
    """
    if not validator.is_type(instance, "object"):
        return

    for prop in instance:
        if prop in disallowed:
            msg = disallowed[prop]
            yield jsonschema.ValidationError(msg)


def union(
    validator: jsonschema.Draft7Validator, det_one_of: Dict, instance: Any, schema: Dict
) -> Iterator[jsonschema.ValidationError]:
    """
    union is for custom error messages with union types.  The built-in oneOf keyword has the same
    validation behavior but awful error handling.  If you had the following invalid hyperparameter:

        hyperparameters:
          - learning_rate:
              type: double
              min: 0.001
              max: 0.005

    would you return an error saying:

        "your double hparam has invalid fields 'min' and 'max' but needs 'minval' and 'maxval'",

    or would you say:

        "your int hparam has type=double but needs type=int and 'minval' and 'maxval'"?

    Obviously you want the first option, because we treat the "type" key as special, and we can
    uniquely identify which subschema should match against the provided data based on the "type".

    The union extension provides this exact behavior.

    Example: The "additionalProperties" schema for the hyperparameters dict:

        "union": {
            "items": [
                {
                    "unionKey": "const:type=int",
                    "$ref": ...
                },
                {
                    "unionKey": "const:type=double",
                    "$ref": ...
                },
                ...
            ]
        }

    When the oneOf validation logic is not met, the error chosen is based on the first unionKey to
    evaluate to true.  In this case, the "const:" means a certain key ("type") must match a certain
    value ("int" or "double") for that subschema's error message to be chosen.
    """
    selected_errors = None
    valid = []

    for idx, item in enumerate(det_one_of["items"]):
        errors = list(validator.descend(instance, schema=item, schema_path=idx))
        if errors:
            key = item["unionKey"]
            if not selected_errors and _evaluate_unionKey(key, instance):
                selected_errors = errors
        else:
            valid.append(item)

    if len(valid) == 1:
        # No errors.
        return

    if len(valid) > 1:
        yield jsonschema.ValidationError(f"bug in validation! Multiple schemas matched: {valid}")
        return

    if selected_errors:
        yield from selected_errors
        return

    default_message = det_one_of.get("defaultMessage", "union failed to validate")
    yield jsonschema.ValidationError(default_message)


def _evaluate_unionKey(key: str, instance: Any) -> bool:
    """
    unionKey is part of the union extension.  It allows for concisely describing when an instance
    of data "should" match a given portion of a subschema of a union type, even when it doesn't
    fully match.  unionKey allows us to select the correct error message to show to the user from
    the union type.
    """
    if key is None:
        return False

    if isinstance(key, str):
        if key == "always":
            return True

        if key == "never":
            return False

        # All other valid keys have arguments.
        key, arg = key.split(":", 1)
        if key == "not":
            return not _evaluate_unionKey(arg, instance)

        if key == "const":
            # "const:NAME=VALUE" returns True when the instance has NAME and it evalutes to VALUE.
            name, value = arg.split("=", 1)
            if not isinstance(instance, dict):
                return False
            return instance.get(name) == value

        if key == "singleproperty":
            # "singleproperty:ATTR" returns True when the instance has ATTR as its only key.
            if not isinstance(instance, dict):
                return False
            if len(instance) != 1:
                return False
            return arg in instance

        if key == "type":
            # "type:TYPE" returns True when the instance's json type is TYPE.
            assert arg in ["array", "object"]
            if arg == "array":
                return isinstance(instance, list)
            if arg == "object":
                return isinstance(instance, dict)

        if key == "hasattr":
            # hasattr:ATTR returns True when the instance has the attribute ATTR.
            return isinstance(instance, dict) and arg in instance

    raise ValueError(f"invalid unionKey: {key}")


def checks(
    validator: jsonschema.Draft7Validator, checks: Dict, instance: Any, schema: Dict
) -> Iterator[jsonschema.ValidationError]:
    """
    checks is a simple extension that returns a specific error if a subschema fails to match.

    The keys of the "checks" dictionary are the user-facing messages, and the values are the
    subschemas that must match.

    Example:

        "checks": {
            "you must specify an entrypoint that references the trial class":{
                ... (schema which allows Native API or requires that entrypoint is set) ...
            },
            "you requested a bayesian search but hyperband is way better": {
                ... (schema which checks if you try searcher.name=baysian) ...
            }
        }
    """
    for msg, subschema in schema["checks"].items():
        errors = list(validator.descend(instance, schema=subschema))
        if errors:
            yield jsonschema.ValidationError(msg)


def compareProperties(
    validator: jsonschema.Draft7Validator, compare: Dict, instance: Any, schema: Dict
) -> Iterator[jsonschema.ValidationError]:
    """
    compareProperties allows a schema to compare values in the instance against each other.
    Amazingly, json-schema does not have a built-in way to do this.

    Example: ensuring that hyperparmeter minval is less than maxval:

        "compareProperties": {
            "type": "a<b",
            "a": "minval",
            "b": "maxval"
        }
    """
    if not validator.is_type(instance, "object"):
        return

    def get_by_path(path: str) -> Any:
        obj = instance
        for key in path.split("."):
            if not obj:
                return None
            obj = obj.get(key)
        return obj

    a_path = compare["a"]
    a = get_by_path(a_path)

    b_path = compare["b"]
    b = get_by_path(b_path)

    if a is None or b is None:
        return

    typ = compare["type"]

    if typ == "a<b":
        if a >= b:
            yield jsonschema.ValidationError(f"{a_path} must be less than {b_path}")
        return

    if typ == "a<=b":
        if a > b:
            yield jsonschema.ValidationError(f"{a_path} must be less than {b_path}")
        return

    if typ == "a_is_subdir_of_b":
        a_norm = os.path.normpath(a)
        b_norm = os.path.normpath(b)
        if os.path.isabs(a_norm):
            if not a_norm.startswith(b_norm):
                yield jsonschema.ValidationError(f"{a_path} must be a subdirectory of {b_path}")
        else:
            if a_norm.startswith(".."):
                yield jsonschema.ValidationError(f"{a_path} must be a subdirectory of {b_path}")
        return

    raise ValueError(f"unrecognized comparison {compare[typ]}")


def eventuallyRequired(
    validator: jsonschema.Draft7Validator,
    eventuallyRequired: Any,
    instance: List,
    schema: Dict,
) -> Iterator[jsonschema.ValidationError]:
    """
    eventuallyRequred allows for two-step validation.  This is a requirement specific to Determined
    because there are fields required (checkpoint_storage) but which may not be present in what the
    user actually submits, since a cluster default may be present.

    eventuallyRequired behaves identically to required, only when building the validator, it is
    possible to not include the eventuallyRequired extension; making it possible to *not* enforce
    eventuallyRequired at specific times.
    """
    for key in eventuallyRequired:
        if key not in instance:
            yield jsonschema.ValidationError(f"{key} is a required property")


def eventually(
    validator: jsonschema.Draft7Validator,
    eventually: Any,
    instance: List,
    schema: Dict,
) -> Iterator[jsonschema.ValidationError]:
    """
    eventually allows for two-step validation, by only enforcing the specified subschemas
    during the completeness validation phase. This is a requirement specific to Determined.

    One use case is when it is necessary to enforce a `oneOf` on two fields that are
    `eventuallyRequired`. If the `oneOf` is evaluated during the sanity validation phase, it will
    always fail, if for example, the user is using cluster default values, but if validation
    for this subschema is held off until completeness validation, it will validate correctly.

    Example: eventually require one of connection string and account url to be specified:

    "eventually": {
        "checks": {
            "Exactly one of connection_string or account_url must be set": {
                "oneOf": [
                    {
                        "eventuallyRequired": [
                            "connection_string"
                        ]
                    },
                    {
                        "eventuallyRequired": [
                            "account_url"
                        ]
                    }
                ]
            }
        }
    }
    """
    yield from validator.descend(instance, schema=eventually, schema_path="eventually")


def optionalRef(
    validator: jsonschema.Draft7Validator, optionalRef: Dict, instance: Any, schema: Dict
) -> Iterator[jsonschema.ValidationError]:
    """
    optionalRef behaves like $ref, except that it also allows the value to be null.

    This is logically equivalent to an anyOf with a {"type": "null"} element, but it has better
    error messages.

    Example: The "internal" property of the experiment config may be a literal null:

        "internal": {
            "type": [
                "object",
                "null"
            ],
            "optionalRef": "http://determined.ai/schemas/expconf/v0/internal.json",
            "default": null
        }
    """
    if instance is None:
        return

    yield from validator.descend(instance, schema={"$ref": optionalRef}, schema_path="optionalRef")
