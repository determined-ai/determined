import os
import re
from typing import Any, Dict, Iterator, List, Optional, Type

import pytest

from determined.common import schemas, yaml
from determined.common.schemas import expconf
from determined.common.schemas.expconf import _v0

yaml = yaml.YAML(typ="safe", pure=True)  # type: ignore


def strip_runtime_defaultable(obj: Any, defaulted: Any) -> Any:
    """
    Recursively find strings of "*" in defaulted and set corresponding non-None values in obj to
    also be "*", so that equality tests will pass.
    """
    if defaulted is None or obj is None:
        return obj

    if isinstance(defaulted, str):
        if defaulted == "*":
            return "*"

    # Recurse through dicts.
    if isinstance(defaulted, dict):
        if not isinstance(obj, dict):
            return obj
        return {k: strip_runtime_defaultable(obj.get(k), defaulted.get(k)) for k in obj}

    if isinstance(defaulted, tuple):
        defaulted = list(defaulted)

    # Recurse through lists.
    if isinstance(defaulted, list):
        if not isinstance(obj, (list, tuple)):
            return obj

        # Truncate defaulted (if it's long) then pad with Nones (if it's short).
        defaulted = defaulted[: len(obj)]
        defaulted = defaulted + [None] * (len(obj) - len(defaulted))

        return [strip_runtime_defaultable(o, d) for o, d in zip(obj, defaulted)]

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
        sane_as: Optional[List[str]] = None,
        complete_as: Optional[List[str]] = None,
        sanity_errors: Optional[Dict[str, str]] = None,
        completeness_errors: Optional[Dict[str, str]] = None,
        defaulted: Any = None,
        default_as: Optional[str] = None,
        merge_as: Optional[str] = None,
        merge_src: Any = None,
        merged: Any = None,
    ) -> None:
        self.name = name
        self.case = case
        self.sane_as = sane_as
        self.complete_as = complete_as
        self.sanity_errors = sanity_errors
        self.completeness_errors = completeness_errors
        self.defaulted = defaulted
        self.default_as = default_as
        self.merge_as = merge_as
        self.merge_src = merge_src
        self.merged = merged

    def run(self) -> None:
        self.run_sanity()
        self.run_completeness()
        self.run_sanity_errors()
        self.run_completeness_errors()
        self.run_defaulted()
        self.run_round_trip()
        self.run_merged()

    def run_sanity(self) -> None:
        if not self.sane_as:
            return
        for url in self.sane_as:
            errors = expconf.sanity_validation_errors(self.case, url)
            if not errors:
                continue
            raise ValueError(f"'{self.name}' failed against {url}:\n - " + "\n - ".join(errors))

    def run_completeness(self) -> None:
        if not self.complete_as:
            return
        for url in self.complete_as:
            errors = expconf.completeness_validation_errors(self.case, url)
            if not errors:
                continue
            raise ValueError(
                ("'{}' failed completeness validation against {}:\n - " "\n - ")
                .format(self.name, url)
                .join(errors)
            )

    def run_sanity_errors(self) -> None:
        if not self.sanity_errors:
            return
        self._run_errors(self.sanity_errors, test_type="sanity")

    def run_completeness_errors(self) -> None:
        if not self.completeness_errors:
            return
        self._run_errors(self.completeness_errors, test_type="completeness")

    def _run_errors(self, error_cases: Any, test_type: str) -> None:
        for url, expected in error_cases.items():
            assert isinstance(expected, list), "malformed test case"
            if test_type == "sanity":
                errors = expconf.sanity_validation_errors(self.case, url)
            else:
                errors = expconf.completeness_validation_errors(self.case, url)
            assert errors, (
                f"'{self.name}' {test_type} validated against {url} unexpectedly, "
                f"expected {expected}"
            )
            for exp in expected:
                for err in errors:
                    if re.search(exp, err):
                        break
                else:
                    msg = f"while testing '{self.name}' for {test_type}, expected to match the"
                    msg += f" pattern\n    {exp}\n"
                    msg += "but it was not found in any of\n    "
                    msg += "\n    ".join(errors)
                    raise ValueError(msg)

    def run_defaulted(self) -> None:
        if not self.defaulted and not self.default_as:
            return
        assert (
            self.defaulted and self.default_as
        ), "`default_as` and `defaulted` must both be present to run a defaulted test"
        cls = class_from_url(self.default_as)

        obj = cls.from_dict(self.case)
        obj.fill_defaults()

        out = obj.to_dict(explicit_nones=True)
        out = strip_runtime_defaultable(out, self.defaulted)
        assert out == self.defaulted, f"failed while testing {self.name}"

    def run_round_trip(self) -> None:
        if not self.defaulted:
            return

        assert self.default_as, "need a `default_as` entry to run a run_round_trip test"
        cls = class_from_url(self.default_as)

        obj0 = cls.from_dict(self.case)

        obj1 = cls.from_dict(obj0.to_dict())
        assert obj1 == obj0, "round trip to_dict/from_dict failed"

        # Round-trip again with defaults.
        obj1.fill_defaults()
        obj2 = cls.from_dict(obj1.to_dict())
        assert obj2 == obj1, "round trip failed with defaults"

    def run_merged(self) -> None:
        if not self.merge_as and not self.merge_src and not self.merged:
            return
        assert (
            self.merge_as and self.merge_src and self.merged
        ), "merge_as, merge_src, and merged must all be present in a test case if any are present"

        # Python expconf doesn't yet support custom merge behavior on list objects, nor does it
        # support the partial checkpoint storage configs.  It probably will never support the
        # partial checkpoint storage (only the master needs it, and hopefully only for the V0
        # schema).
        pytest.skip("python expconf doesn't support merge tests yet")

        cls = class_from_url(self.merge_as)

        obj = cls.from_dict(self.case)
        src = cls.from_dict(self.merge_src)

        merged = schemas.SchemaBase.merge(obj, src)
        assert merged == self.merged, f"failed while testing {self.name}"


CASES_ROOT = os.path.join(os.path.dirname(__file__), "..", "..", "..", "schemas", "test_cases")


# Get a list of all test cases.  They are parameterized this way because it makes it extremely
# easy to identify which test failed from the log output.  Otherwise, we would just yield the full
# test case here.
def all_cases() -> Iterator["str"]:
    for root, _, files in os.walk(CASES_ROOT):
        for file in files:
            if file.endswith(".yaml"):
                path = os.path.join(root, file)
                with open(path) as f:
                    cases = yaml.load(f)
                for case in cases:
                    display_path = os.path.relpath(path, CASES_ROOT)
                    yield display_path + "::" + case["name"]


@pytest.mark.parametrize("test_case", all_cases())
def test_schemas(test_case: str) -> None:
    cases_file, case_name = test_case.split("::", 1)
    with open(os.path.join(CASES_ROOT, cases_file)) as f:
        cases = yaml.load(f)
    # Find the right test from the file of test cases.
    case = [c for c in cases if c["name"] == case_name][0]
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
