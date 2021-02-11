import os
import re
from typing import Any, Dict, Iterator, List, Optional, Type

import pytest

from determined_common import schemas, yaml
from determined_common.schemas import expconf
from determined_common.schemas.expconf import _v0


def strip_runtime_defaultable(obj: Any, defaulted: Any) -> Any:
    """
    Recursively find strings of "*" in defaulted and set corresponding non-None values in obj to
    also be "*", so that equality tests will pass.
    """
    if isinstance(defaulted, str):
        if defaulted == "*" and obj is not None:
            return "*"

    # Recurse through dicts.
    if isinstance(defaulted, dict):
        if not isinstance(obj, dict):
            return
        common_keys = set(defaulted.keys()).intersection(set(obj.keys()))
        return {k: strip_runtime_defaultable(obj.get(k), defaulted.get(k)) for k in common_keys}

    # Recurse through lists.
    if isinstance(defaulted, (list, tuple)):
        if not isinstance(obj, (list, tuple)):
            return
        limit = min(len(obj), len(defaulted))
        return [strip_runtime_defaultable(obj[i], defaulted[i]) for i in range(limit)]

    return obj


def class_from_url(url: str) -> Type[schemas.SchemaBase]:
    schema = expconf.get_schema(url)
    title = schema["title"]

    cls = getattr(_v0, title + "V0")
    assert issubclass(cls, schemas.SchemaBase)
    return cls  # type: ignore


class Case:
    def __init__(
        self,
        name: str,
        case: Any,
        matches: Optional[List[str]] = None,
        errors: Optional[Dict[str, str]] = None,
        defaulted: Any = None,
    ) -> None:
        self.name = name
        self.case = case
        self.matches = matches
        self.errors = errors
        self.defaulted = defaulted

    def run(self) -> None:
        self.run_matches()
        self.run_errors()
        self.run_defaulted()
        self.run_round_trip()

    def run_matches(self) -> None:
        if not self.matches:
            return
        for url in self.matches:
            errors = expconf.validation_errors(self.case, url)
            if not errors:
                continue
            raise ValueError(f"'{self.name}' failed against {url}:\n - " + "\n - ".join(errors))

    def run_errors(self) -> None:
        if not self.errors:
            return
        for url, expected in self.errors.items():
            assert isinstance(expected, list), "malformed test case"
            errors = expconf.validation_errors(self.case, url)
            assert errors, f"'{self.name}' matched {url} unexpectedly"
            for exp in expected:
                for err in errors:
                    if re.search(exp, err):
                        break
                else:
                    msg = f"while testing '{self.name}', expected to match the pattern\n"
                    msg += f"    {exp}\n"
                    msg += "but it was not found in any of\n    "
                    msg += "\n    ".join(errors)
                    raise ValueError(msg)

    def run_defaulted(self) -> None:
        if not self.defaulted:
            return
        assert self.matches, "need a `matches` entry to run a defaulted test"
        cls = class_from_url(self.matches[0])

        obj = cls.from_dict(self.case)
        obj.fill_defaults()

        out = obj.to_dict(explicit_nones=True)
        out = strip_runtime_defaultable(out, self.defaulted)
        assert out == self.defaulted, f"failed while testing {self.name}"

    def run_round_trip(self) -> None:
        if not self.defaulted:
            return

        assert self.matches, "need a `matches` entry to run a run_round_trip test"
        cls = class_from_url(self.matches[0])

        obj0 = cls.from_dict(self.case)

        obj1 = cls.from_dict(obj0.to_dict())
        assert obj1 == obj0, "round trip to_dict/from_dict failed"

        # Round-trip again with defaults.
        obj1.fill_defaults()
        obj2 = cls.from_dict(obj1.to_dict())
        assert obj2 == obj1, "round trip failed with defaults"


CASES_ROOT = os.path.join(os.path.dirname(__file__), "..", "..", "schemas", "test_cases")


# Get a list of all test cases.
def all_cases() -> Iterator["str"]:
    for root, _, files in os.walk(CASES_ROOT):
        for file in files:
            if file.endswith(".yaml"):
                path = os.path.join(root, file)
                with open(path) as f:
                    cases = yaml.safe_load(f)
                for case in cases:
                    display_path = os.path.relpath(path, CASES_ROOT)
                    yield display_path + "::" + case["name"]


@pytest.mark.parametrize("test_case", all_cases())  # type: ignore
def test_schemas(test_case: str) -> None:
    cases_file, case_name = test_case.split("::", 1)
    with open(os.path.join(CASES_ROOT, cases_file)) as f:
        cases = yaml.safe_load(f)
    for case in cases:
        Case(**case).run()


def lint_schema_subclasses(cls: type) -> None:
    """Recursively check all SchemaBase subclasses"""
    for sub in cls.__subclasses__():
        lint_schema_subclasses(sub)

    # Ignore certain classes.
    if cls in [schemas.SchemaBase]:
        return

    # class annotations should match __init__ arg annotations.
    cls_annos = {
        (name, anno) for name, anno in cls.__annotations__.items() if not name.startswith("_")
    }
    init_annos = {
        (name, anno)
        for name, anno in cls.__init__.__annotations__.items()  # type: ignore
        if name != "return"
    }
    assert (
        cls_annos == init_annos
    ), f"{cls.__name__}: class annotions and __init__ args do not match"

    defaults = [name for name, _ in cls_annos if hasattr(cls, name)]

    # All default values should be None.
    for name in defaults:
        assert getattr(cls, name) is None, (
            f"{cls.__name__}.{name} must default to None; schema objects are like typed "
            "dictionaries and None is the value to represent that no other value was specified. "
            "Defaults from the json-schema definitions are applied via .fill_defaults()"
        )

    # class values should match __init__ arg defaults.
    cls_defaults = {(name, getattr(cls, name)) for name in defaults}
    init_defaults = {
        (name, (cls.__init__.__defaults__ or {}).get(name)) for name in defaults  # type: ignore
    }
    assert (
        cls_defaults == init_defaults
    ), f"{cls.__name__}: class defaults and __init__ defaults do not match"


def test_schema_class_definitons() -> None:
    lint_schema_subclasses(schemas.SchemaBase)
