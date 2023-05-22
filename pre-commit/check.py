#!/usr/bin/env python

import argparse
import concurrent.futures
import multiprocessing
import os
import subprocess
import sys
from pathlib import Path
from typing import Dict, Iterable, List, Set, Tuple, Union

# TODO add option to pass in a git diff to check or git hash


def print_colored(skk, **kwargs):
    print("\033[93m {}\033[00m".format(skk), **kwargs)


# check if path a child of another path
def is_child(path: Path, parent: Path) -> bool:
    assert path.is_absolute()
    assert parent.is_absolute()
    cur_path = path
    while cur_path != parent and cur_path != root:
        cur_path = cur_path.parent
        if cur_path == parent:
            return True
    return False


if os.environ.get("PRE_COMMIT_ENABLE_SLOW", "false") != "true":
    print("Skipping slow checks.")
    print("Set PRE_COMMIT_ENABLE_SLOW=true to activate.")
    exit(0)

PROJECT_NAME = "determined"
root = Path(os.getenv("PROJECT_ROOT", os.getcwd())).absolute()
if not str(root).endswith(PROJECT_NAME):
    print(
        "Please run this script from the root of the project or set it using "
        + "PROJECT_ROOT env variable"
    )
    exit(1)
os.chdir(root)

# rules are a mapping of different modules to their relative command(s) to run.
# we might want to separate or support nested modules later on.
rules: Dict[Path, Union[str, List[str]]] = {
    root
    / "harness": [
        "make -j fmt; make -j check",
        "make -j build",
    ],
    root
    / "proto": [
        "make fmt check build",
        "make -C ../bindings build check",
    ],
    # root / "webui" / "react": ["make -j fmt; make -j check", "make -j test && make -j build"], # mostly covered by proper precommit checks
    root
    / "master": [
        "make build",
        "make fmt check",
    ],
    root / "docs": "make fmt check build",
    root / ".circleci": "type circleci || exit 0 && circleci config validate config.yml",
    root / "e2e_tests": "make fmt check",
    root / "model_hub": "make fmt check",
}


def run_rule(rule_path: Path) -> Tuple[int, str]:
    rule = rules[rule_path]
    cmds = rule if isinstance(rule, list) else [rule]
    for cmd in cmds:  # run commands sequentially with early breaking.
        assert isinstance(cmd, str)
        try:
            subprocess.run(
                cmd,
                cwd=rule_path,
                shell=True,
                check=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
            )
        except subprocess.CalledProcessError as e:
            return e.returncode, e.stderr.decode("utf-8")
    return (0, "")


def find_rules(paths: List[Path]):
    resolved_paths: Set[Path] = set()
    for dirty_path in paths:
        for rule_path in rules.keys():
            if is_child(dirty_path, rule_path):
                resolved_paths.add(rule_path)
    return resolved_paths


def report_result(rule_path: Path, return_code: int, msg: str):
    """report the results of running a rule/check."""
    if return_code != 0:
        print_colored(f"{rule_path.relative_to(root)}: failed", file=sys.stderr)
        print(msg, file=sys.stderr)
    else:
        print(f"{rule_path.relative_to(root)}: passed")


def process_rules(module_paths: Iterable[Path]) -> Set[Tuple[Path, str]]:
    failed_rules: Set[Tuple[Path, str]] = set()
    # there are some rule dependencies that we need to run sequentially. eg proto and master
    max_workers = 1  # multiprocessing.cpu_count()
    with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
        # collect the future to args mapping so we can print module path.
        future_to_rule = {
            executor.submit(run_rule, rule_path): rule_path for rule_path in module_paths
        }
        for future in concurrent.futures.as_completed(future_to_rule):
            rule_path = future_to_rule[future]
            try:
                return_code, msg = future.result()
                report_result(rule_path, return_code, msg)
                if return_code != 0:
                    failed_rules.add((rule_path, msg))
            except Exception as exc:
                print(f"{rule_path} generated an exception: {exc}", file=sys.stderr)
    return failed_rules


def main():
    argparser = argparse.ArgumentParser()
    argparser.add_argument("files", nargs="*", type=str, help="files to check")
    args = argparser.parse_args()

    changed_files = [Path(x).absolute() for x in args.files]
    if len(changed_files) == 0:
        print("No changed files.")
        exit(0)

    failed_rules = process_rules(find_rules(changed_files))
    if len(failed_rules):
        print_colored(f"{len(failed_rules)} check(s) failed.", file=sys.stderr)

        exit(1)


main()
