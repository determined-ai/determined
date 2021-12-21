#!/usr/bin/env python3
#
# Sphinx-build wrapper
#
# Causes non-zero exit code when there are broken download links
#
import os
import re
import sys
import subprocess
import tempfile

from typing import List

# "sphinx-build -W" does not seem to stop on the following warning:
# download file not readable: docs/site/downloads/helm/determined-0.5.0.tgz
download_file_regex = re.compile(r"download file not readable: (?P<filepath>.+)$")


def main(sphinx_build_args: List[str]) -> int:
    if sys.stdout.isatty():
        sphinx_build_args.append("--color")

    download_link_errors = []
    with tempfile.TemporaryDirectory() as temp_dir:
        stderr_path = os.path.join(temp_dir, "sphinx-build.stderr")

        redirect_args = (
            f"{' '.join(sphinx_build_args)} 3>&1 1>&2 2>&3 | tee {stderr_path}"
        )
        process = subprocess.Popen(redirect_args, shell=True)
        process.wait()

        # Read stderr
        with open(stderr_path, "r") as stderr:
            for line in stderr.readlines():
                m = re.match(download_file_regex, line)
                if m is not None:
                    download_link_errors.append(m.group("filepath"))

    if process.returncode != 0:
        sys.stderr.write(f"sphinx-build exited non-zero({process.returncode})\n")
        return process.returncode

    if len(download_link_errors) == 0:
        return 0

    sys.stderr.write(
        "Download links are broken!! ':download:`<path> links`'\n" "Full paths:\n"
    )
    for path in download_link_errors:
        sys.stderr.write(f"{path}\n")

    sys.stderr.write(
        "Please grep the source for each filename in the docs "
        "and update the docs with the correct file names.\n"
    )

    return 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
