import os
from typing import Any, Dict, Iterator, List, Optional

from determined_common import yaml
from determined_common.schemas import expconf

test_cases_path = os.path.join(os.path.dirname(__file__), "..", "..", "schemas", "test_cases", "v1")


def cases_files() -> Iterator["str"]:
    for case_file in os.listdir(test_cases_path):
        if case_file.endswith(".yaml"):
            yield os.path.join(test_cases_path, case_file)


class Case:
    def __init__(
        self,
        name: str,
        case: Any,
        matches: Optional[List[str]] = None,
        errors: Optional[Dict[str, str]] = None,
    ) -> None:
        self.name = name
        self.case = case
        self.matches = matches
        self.errors = errors

    def run(self) -> None:
        self.run_matches()
        self.run_errors()

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
            errors = expconf.validation_errors(self.case, url)
            assert errors, f"'{self.name}' matched {url} unexpectedly"
            for exp in expected:
                for err in errors:
                    if exp in err:
                        break
                else:
                    msg = f"while testing '{self.name}', expected to see\n"
                    msg += f"    {exp}\n"
                    msg += "but it was not found in any of\n    "
                    msg += "\n    ".join(errors)
                    raise ValueError(msg)


def test_v1() -> None:
    for cases_file in cases_files():
        with open(cases_file) as f:
            cases = yaml.safe_load(f)
        for case in cases:
            Case(**case).run()
