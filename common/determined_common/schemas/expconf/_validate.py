from typing import Any, Dict, List, Optional

import jsonschema

from determined_common.schemas import extensions, util
from determined_common.schemas.expconf import _v1_gen

_v1_validators = {}  # type: Dict[str, Any]


def v1_validator(url: Optional[str] = None) -> Any:
    # Use the experiment config schema by default.
    if url is None:
        url = "http://determined.ai/schemas/expconf/v1/experiment.json"

    global _v1_validators
    if url in _v1_validators:
        return _v1_validators[url]

    schema = _v1_gen.schemas[url]

    resolver = jsonschema.RefResolver(
        base_uri=url,
        referrer=schema,
        handlers={"http": lambda url: _v1_gen.schemas[url]},
    )

    validator = jsonschema.Draft7Validator(schema=schema, resolver=resolver)
    ext = {
        "disallowProperties": extensions.disallowProperties,
        "union": extensions.union,
        "checks": extensions.checks,
        "compareProperties": extensions.compareProperties,
        "conditional": extensions.conditional,
        "eventuallyRequired": extensions.eventuallyRequired,
    }
    cls = jsonschema.validators.extend(validator, ext)
    _v1_validators[url] = cls(schema=schema, resolver=resolver)

    return _v1_validators[url]


def validation_errors(instance: Any, url: Optional[str] = None) -> List[str]:
    validator = v1_validator(url)
    errors = validator.iter_errors(instance)
    return util.format_validation_errors(errors)
