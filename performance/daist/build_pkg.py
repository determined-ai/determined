#!/usr/bin/env python3

import shlex
import subprocess

from daist.framework import repo, venv
from daist.framework.paths import RootPath, VenvPath

if __name__ == '__main__':
    venv.make_for_build()
    cmd = f'{VenvPath.PYTHON} -m build --outdir {RootPath.DIST_OUTPUT}'
    subprocess.run(shlex.split(cmd), check=True, env=repo.get_build_env())
