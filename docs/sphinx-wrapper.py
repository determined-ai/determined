#!/usr/bin/env python3
#
# Sphinx-build wrapper
#
# Causes non-zero exit code when there are broken download links
#
import re
import sys
import subprocess

from typing import List
from itertools import zip_longest

# download file not readable: docs/site/downloads/helm/determined-0.5.0.tgz
download_file_regex = re.compile(
    r'download file not readable: (?P<filepath>.+)$')


def main(sphinx_build_args: List[str]) -> int:
    process = subprocess.Popen(
        [*sphinx_build_args],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE)

    download_link_errors = []

    out_err = zip_longest(iter(process.stdout.readline, b''),
                          iter(process.stderr.readline, b''))

    for out_bytes, err_bytes in out_err:
        if out_bytes is not None:
            try:
                out = out_bytes.decode('utf-8')
                sys.stdout.write(out)
            except UnicodeDecodeError:
                sys.stdout.write(out_bytes)

        if err_bytes is not None:
            try:
                err = err_bytes.decode('utf-8')
                sys.stderr.write(err)
            except UnicodeDecodeError:
                sys.stderr.write(err_bytes)

            m = re.match(download_file_regex, err)
            if m is not None:
                download_link_errors.append(m.group('filepath'))

    process.wait()

    if process.returncode != 0:
        sys.stderr.write("sphinx-build exited non-zero({process.returncode})")
        return process.returncode

    if len(download_link_errors) == 0:
        return 0

    sys.stderr.write("Download links are broken!! ':download:`<path> links`'\n"
                     "Full paths:\n")
    for path in download_link_errors:
        sys.stderr.write(f"{path}\n")

    sys.stderr.write("Please grep the source for each filename in the docs "
                     "and update the docs with the correct file names.\n")

    return 1


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
