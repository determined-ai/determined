#!/usr/bin/env python

from typing import List, Union, Dict, Set
from pathlib import Path
import argparse
import os
import sys

# TODO add option to run rules in parallel
# TODO add option to pass in a git diff to check or git hash


def get_git_commit_files(commit_hash: str) -> List[Path]:
    output = os.popen(f"git diff-tree --no-commit-id --name-only -r {commit_hash}").read()
    lines = output.split("\n")
    lines = [x for x in lines if x]
    files = [Path(x).absolute() for x in lines]
    return files


# get a list of paths to dirty and staged files from git
def get_git_status() -> List[Path]:
    git_status = os.popen("git status --porcelain").read()
    git_status_list = git_status.split("\n")
    git_status_list = [x for x in git_status_list if x]
    files = [Path(x.split()[1]).absolute() for x in git_status_list]
    return files


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


# check if PRECOMMIT_ENABLE_SLOW is set to true
if os.environ.get("PRE_COMMIT_ENABLE_SLOW", "false") != "true":
    print("Skipping slow checks.")
    print("Set PRE_COMMIT_ENABLE_SLOW=true to activate.")
    exit(0)

PROJECT_NAME = "determined"
root = Path(os.getenv("PROJECT_ROOT", os.getcwd())).absolute()
if not str(root).endswith(PROJECT_NAME) and not str(root).endswith("saas"):  # FIXME
    print(
        "Please run this script from the root of the project or set it using "
        + "PROJECT_ROOT env variable"
    )
    exit(1)
os.chdir(root)

rules: Dict[Path, Union[str, List[str]]] = {
    root
    / "harness": [
        "make -j fmt; make -j check",
        "make -j build",
    ],
    root
    / "proto": [
        "make fmt check build",
        "make -C ../bindings build && make -C ../webui/react bindings-copy-over && make -C ../webui/react check",
    ],
    root / "webui" / "react": ["make -j fmt; make -j check", "make -j test && make -j build"],
    root
    / "master": [
        "make -C ../proto build",
        "make build",
        "make fmt; make check",
        "make test",
    ],
    root / "docs": "make fmt check build",
    root / ".circleci": "circleci config validate config.yml",
    root / "e2e_tests": "make fmt check",
    root / "model_hub": "make fmt check",
}


# check if path is the same or child of one of the rules and execute the rule as
# subprocess
def run_rule(rule_path: Path) -> int:
    rule = rules[rule_path]
    os.chdir(rule_path)
    print(f'in direcotry "{rule_path.relative_to(root)}" run: {rule}')
    cmds = rule if isinstance(rule, list) else [rule]
    for cmd in cmds:
        assert isinstance(cmd, str)
        return_code = os.system(cmd)
        if return_code != 0:
            print(f'command "{cmd}" failed with return code {return_code}', file=sys.stderr)
            return return_code
    return 0


def find_rules(paths: List[Path]):
    resolved_paths = set()
    for dirty_path in paths:
        for rule_path in rules.keys():
            if is_child(dirty_path, rule_path):
                resolved_paths.add(rule_path)
    return resolved_paths


def main():
    argparser = argparse.ArgumentParser()
    argparser.add_argument("files", nargs="*", type=str, help="files to check")
    args = argparser.parse_args()

    changed_files = [Path(x).absolute() for x in args.files]
    if len(changed_files) == 0:
        print("No changed files.")
        exit(0)

    failed_rules: Set[Path] = set()
    for rule_path in find_rules(changed_files):
        rv = run_rule(rule_path)
        if rv != 0:
            failed_rules.add(rule_path)
    if len(failed_rules):
        print(
            f"{len(failed_rules)} check(s) failed {[str(r) for r in failed_rules]}", file=sys.stderr
        )
        exit(1)


main()
