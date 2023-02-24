#!/usr/bin/env python3

import argparse
import multiprocessing
import os
import subprocess


def web_lint_check():
    parser = argparse.ArgumentParser(description="Lint Check for Web")
    parser.add_argument("target", help="Either js, css, or misc", choices=["js", "css", "misc"])
    parser.add_argument("file_paths", help="1 or more file paths", nargs="+")
    args = parser.parse_args()
    DIR = "webui/react/"

    target: str = args.target
    file_paths: str = " ".join([os.path.relpath(file_path, DIR) for file_path in args.file_paths])
    nproc: int = multiprocessing.cpu_count()
    run_command: list[str] = [
        "make",
        f"-j{nproc}",
        "-C",
        DIR,
        "prettier",
        f"PRE_ARGS=-- --write -c {file_paths}",
    ]

    # TODO: replace it with `match` if we support python v3.10
    if target == "js":
        run_command += ["eslint", f"ES_ARGS=-- --fix {file_paths}"]
    elif target == "css":
        run_command += ["stylelint", f"ST_ARGS=-- --fix {file_paths}"]
    elif target == "misc":
        run_command += ["check-package-lock"]

    returncode: int = subprocess.call(run_command)
    exit(returncode)


if __name__ == "__main__":
    web_lint_check()
