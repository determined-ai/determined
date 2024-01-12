import os
import pathlib
from typing import TYPE_CHECKING, Callable, Iterable, List, Set

if TYPE_CHECKING:
    import pathspec

from determined.common import constants, v1file_utils
from determined.common.api import bindings


def _build_detignore_pathspec(root_path: pathlib.Path) -> "pathspec.PathSpec":
    ignore = list(constants.DEFAULT_DETIGNORE)
    ignore_path = root_path / ".detignore"
    if ignore_path.is_file():
        with ignore_path.open("r") as detignore_file:
            ignore.extend(detignore_file)

    # Lazy import to speed up load time.
    # See https://github.com/determined-ai/determined/pull/6590 for details.
    import pathspec

    return pathspec.PathSpec.from_lines(pathspec.patterns.GitWildMatchPattern, ignore)


def make_shutil_ignore(root_path: pathlib.Path) -> Callable:
    ignore_spec = _build_detignore_pathspec(root_path)

    def _ignore(path: str, names: List[str]) -> Set[str]:
        ignored_names = set()  # type: Set[str]
        for name in names:
            if name == ".detignore":
                ignored_names.add(name)
                continue

            file_path = pathlib.Path(path) / name
            file_rel_path = file_path.relative_to(root_path)

            if (
                file_path.is_dir() and ignore_spec.match_file(str(file_rel_path) + "/")
            ) or ignore_spec.match_file(str(file_rel_path)):
                ignored_names.add(name)

        return ignored_names

    return _ignore


def os_walk_to_v1Files(
    root_path: pathlib.Path, entry_prefix: pathlib.Path
) -> Iterable[bindings.v1File]:
    ignore_spec = _build_detignore_pathspec(root_path)

    for parent, dirs, files in os.walk(str(root_path)):
        keep_dirs = []
        for directory in dirs:
            dir_path = pathlib.Path(parent).joinpath(directory)
            dir_rel_path = dir_path.relative_to(root_path)

            # If the directory matches any path specified in .detignore, then ignore it.
            if ignore_spec.match_file(str(dir_rel_path)):
                continue
            if ignore_spec.match_file(str(dir_rel_path) + "/"):
                continue
            keep_dirs.append(directory)
            # Determined only supports POSIX-style file paths.  Use as_posix() in case this code
            # is executed in a non-POSIX environment.
            entry_path = (entry_prefix / dir_rel_path).as_posix()

            yield v1file_utils.v1File_from_local_dir(entry_path, dir_path)
        # We can modify dirs in-place so that we do not recurse into ignored directories
        #  See https://docs.python.org/3/library/os.html#os.walk
        dirs[:] = keep_dirs

        for file in files:
            file_path = pathlib.Path(parent).joinpath(file)
            file_rel_path = file_path.relative_to(root_path)

            # If the file is the .detignore file or matches one of the
            # paths specified in .detignore, then ignore it.
            if file_rel_path.name == ".detignore":
                continue
            if ignore_spec.match_file(str(file_rel_path)):
                continue

            # Determined only supports POSIX-style file paths.  Use as_posix() in case this code
            # is executed in a non-POSIX environment.
            entry_path = (entry_prefix / file_rel_path).as_posix()

            try:
                entry = v1file_utils.v1File_from_local_file(entry_path, file_path)
            except OSError:
                print(f"Error reading '{entry_path}', skipping this file.")
                continue

            yield entry
