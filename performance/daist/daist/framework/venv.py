import os
import shlex
import subprocess
import sys

from .paths import RootPath, VenvPath


class Env:
    GIT_COMMIT = 'GIT_COMMIT'


def activate():
    if not os.path.samefile(sys.prefix, VenvPath.PATH):
        os.execvp(VenvPath.PYTHON, [VenvPath.PYTHON] + sys.argv)


def make():
    if not VenvPath.PATH.exists():
        _create()
        subprocess.run(shlex.split(f'{VenvPath.PIP} install -e {RootPath.PATH}'))


def make_for_build():
    if not VenvPath.PATH.exists():
        _create()
    subprocess.run(shlex.split(f'{VenvPath.PIP} install -e {RootPath.PATH}[BUILD]'), check=True)


def _create():
    subprocess.run(shlex.split(f'python -m venv {VenvPath.PATH} --clear --system-site-packages '
                               f'--upgrade-deps'), check=True)
