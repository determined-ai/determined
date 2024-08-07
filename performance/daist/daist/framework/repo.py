import os
import shlex
import subprocess
from importlib import metadata
from typing import Optional, Tuple

import daist

from ..models.environment import environment
from .paths import RootPath

DIRTY_TAG = '.dirty'
VERSION_COMMIT_DELIMITER = '+'


def get_build_env() -> dict:
    env = dict(os.environ)
    git_commit = get_commit()
    env[environment.Key.GIT_COMMIT] = git_commit
    return env


def get_branch() -> Optional[str]:
    branch = environment.git_branch
    if branch is None:
        proc = subprocess.run(shlex.split('git rev-parse --abbrev-ref HEAD'), check=False,
                              stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, encoding='utf-8')
        if proc.returncode == 0:
            branch = proc.stdout
        else:
            branch = None
    return branch.strip()


def get_commit(mark_dirty_if_needed: bool = True) -> str:
    commit = environment.git_commit
    if commit is None:
        proc = subprocess.run(shlex.split('git rev-parse HEAD'), check=False,
                              stdout=subprocess.PIPE,
                              stderr=subprocess.DEVNULL, encoding='utf-8')
        if proc.returncode == 0:
            commit = proc.stdout
        else:
            commit = metadata.version(RootPath.PKG.name).split(VERSION_COMMIT_DELIMITER)[-1]

    commit = commit.strip()

    if is_repo_dirty() and mark_dirty_if_needed:
        commit += DIRTY_TAG

    return commit


def is_repo_dirty() -> bool:
    """
    Inspired by: https://stackoverflow.com/a/2659808/1144204
    """
    is_dirty = False

    proc = subprocess.run(shlex.split('git diff-index --quiet --exit-code HEAD --'),
                          check=False)
    is_dirty |= bool(proc.returncode)

    proc = subprocess.run(shlex.split('git ls-files --exclude-standard --others'),
                          stdout=subprocess.PIPE, encoding='utf-8', check=True)
    is_dirty |= bool(proc.stdout)

    return is_dirty


def parse_version() -> Tuple[str, str, bool]:
    version, commit = daist.__version__.split(VERSION_COMMIT_DELIMITER)
    is_dirty = commit.endswith(DIRTY_TAG)

    if is_dirty:
        commit = commit.split(DIRTY_TAG)[0]

    return version, commit, is_dirty
