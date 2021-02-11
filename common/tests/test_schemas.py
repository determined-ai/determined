import os
import re
from typing import Any, Dict, Iterator, List, Optional

from determined_common import yaml
from determined_common.schemas import expconf

test_cases_path = os.path.join(os.path.dirname(__file__), "..", "..", "schemas", "test_cases")


def cases_files() -> Iterator["str"]:
    for root, _, files in os.walk(test_cases_path):
        for file in files:
            if file.endswith(".yaml"):
                yield os.path.join(root, file)


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
        url = self.matches[0]

        schema = expconf.get_schema(url)
        title = schema["title"]
        from determined_common.schemas.expconf import _v0

        cls = getattr(_v0, title + "V0")

        obj = cls.from_dict(self.case)
        obj.fill_defaults()

        out = obj.to_dict(explicit_nones=True)
        out = strip_runtime_defaultable(out, self.defaulted)
        assert out == self.defaulted, f"failed while testing {self.name}"

    def run_round_trip(self) -> None:
        if not self.defaulted:
            return
        # TODO: finish this


def test_schemas() -> None:
    for cases_file in cases_files():
        with open(cases_file) as f:
            cases = yaml.safe_load(f)
        for case in cases:
            Case(**case).run()
