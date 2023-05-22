#!/usr/bin/env python3
#
# Sphinx-build wrapper
#
# Causes non-zero exit code when there are broken download links
#
import os
import re
import subprocess
import sys
import tempfile
from typing import List

# "sphinx-build -W" does not seem to stop on the following warning:
# download file not readable: docs/site/downloads/helm/determined-0.5.0.tgz
download_file_regex = re.compile(b"download file not readable: (?P<filepath>.+)$")


def main(sphinx_build_args: List[str]) -> int:
    p = subprocess.Popen(sphinx_build_args, stderr=subprocess.PIPE)

    # Parse stderr for link errors while also passing it through to stdout.
    download_link_errors = []
    for line in p.stderr:
        os.write(sys.stdout.fileno(), line)
        m = re.match(download_file_regex, line)
        if m is not None:
            download_link_errors.append(m.group("filepath"))

    ret = p.wait()
    if ret != 0:
        return ret

    if len(download_link_errors) == 0:
        return 0

    sys.stderr.write("Download links are broken!! ':download:`<path> links`'\n" "Full paths:\n")
    for path in download_link_errors:
        sys.stderr.write(f"{path}\n")

    sys.stderr.write(
        "Please grep the source for each filename in the docs "
        "and update the docs with the correct file names.\n"
    )

    return 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
