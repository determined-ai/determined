#!/usr/bin/env python

from typing import Iterable, List, Tuple, Union, Dict, Set
from pathlib import Path
import multiprocessing
import subprocess
import argparse
import os
import sys
import concurrent.futures

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
        "make -C ../bindings build && make -C ../webui/react bindings-copy-over && make -C ../webui/react check",
    ],
    # root / "webui" / "react": ["make -j fmt; make -j check", "make -j test && make -j build"], # mostly covered by proper precommit checks
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


def run_rule(rule_path: Path) -> Tuple[int, str]:
    rule = rules[rule_path]
    cmds = rule if isinstance(rule, list) else [rule]
    for cmd in cmds:  # run commands sequentially with early breaking.
        assert isinstance(cmd, str)
        proc = subprocess.run(
            cmd,
            cwd=rule_path,
            shell=True,
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        if proc.returncode != 0:
            return proc.returncode, proc.stderr.decode("utf-8")
    return (0, "")


def find_rules(paths: List[Path]):
    resolved_paths: Set[Path] = set()
    for dirty_path in paths:
        for rule_path in rules.keys():
            if is_child(dirty_path, rule_path):
                resolved_paths.add(rule_path)
    return resolved_paths


def process_rules(module_paths: Iterable[Path]) -> Set[Tuple[Path, str]]:
    failed_rules: Set[Tuple[Path, str]] = set()
    with concurrent.futures.ThreadPoolExecutor(max_workers=multiprocessing.cpu_count()) as executor:
        # collect the future to args mapping so we can print module path.
        future_to_rule = {
            executor.submit(run_rule, rule_path): rule_path for rule_path in module_paths
        }
        for future in concurrent.futures.as_completed(future_to_rule):
            rule_path = future_to_rule[future]
            try:
                return_code, cmd = future.result()
                if return_code != 0:
                    failed_rules.add((rule_path, cmd))
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
        print(
            f"{len(failed_rules)} check(s) failed {[str(r) for r in failed_rules]}", file=sys.stderr
        )
        exit(1)


main()
