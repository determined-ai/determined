import base64
import collections
import os
import pathlib
import tarfile
from typing import Any, Dict, Iterable, List, Optional

import pathspec

from determined.common import constants
from determined.common.api import bindings
from determined.common.util import sizeof_fmt

LegacyContext = List[Dict[str, Any]]


def v1File_size(f: bindings.v1File) -> int:
    if f.content:
        # The content is already base64-encoded, and we want the real length.
        return len(f.content) // 4 * 3
    return 0


def v1File_to_dict(f: bindings.v1File) -> Dict[str, Any]:
    d = {
        "path": f.path,
        "type": f.type,
        "uid": f.uid,
        "gid": f.gid,
        # Echo API expects int-type int64 value
        "mtime": int(f.mtime),
        "mode": f.mode,
    }
    if f.type in (ord(tarfile.REGTYPE), ord(tarfile.DIRTYPE)):
        d["content"] = f.content
    return d


def v1File_from_local_file(archive_path: str, path: pathlib.Path) -> bindings.v1File:
    with path.open("rb") as f:
        content = base64.b64encode(f.read()).decode("utf8")
    st = path.stat()
    return bindings.v1File(
        path=archive_path,
        type=ord(tarfile.REGTYPE),
        content=content,
        # Protobuf expects string-encoded int64
        mtime=str(int(st.st_mtime)),
        mode=st.st_mode,
        uid=0,
        gid=0,
    )


def v1File_from_local_dir(archive_path: str, path: pathlib.Path) -> bindings.v1File:
    st = path.stat()
    return bindings.v1File(
        path=archive_path,
        type=ord(tarfile.DIRTYPE),
        content="",
        # Protobuf expects string-encoded int64
        mtime=str(int(st.st_mtime)),
        mode=st.st_mode,
        uid=0,
        gid=0,
    )


class _Builder:
    def __init__(self, limit: int) -> None:
        self.limit = limit
        self.size = 0
        self.items = []  # type: List[bindings.v1File]
        self.msg = f"Preparing files to send to master... {sizeof_fmt(0)} and 0 files"
        print(self.msg, end="\r", flush=True)

    def add_v1File(self, f: bindings.v1File) -> None:
        self.items.append(f)
        self.size += v1File_size(f)
        if self.size > self.limit:
            raise ValueError(
                "The total size of context directory and included files and directories exceeds "
                f" the maximum allowed size {sizeof_fmt(self.limit)}.\n"
                "Consider using either .detignore files inside directories to specify that certain "
                "files or subdirectories should be omitted."
            )

    def update_msg(self) -> None:
        print(" " * len(self.msg), end="\r")
        self.msg = (
            "Preparing files to send to master... "
            f"{sizeof_fmt(self.size)} and {len(self.items)} files"
        )
        print(self.msg, end="\r", flush=True)

    def add(self, root_path: pathlib.Path, entry_prefix: pathlib.Path) -> None:
        root_path = root_path.resolve()

        if not root_path.exists():
            raise ValueError(f"Path '{root_path}' doesn't exist")

        if root_path.is_file():
            self.add_v1File(v1File_from_local_file(root_path.name, root_path))
            return

        if str(entry_prefix) != ".":
            # For non-context directories, include the root directory.
            self.add_v1File(v1File_from_local_dir(str(entry_prefix), root_path))

        ignore = list(constants.DEFAULT_DETIGNORE)
        ignore_path = root_path.joinpath(".detignore")
        if ignore_path.is_file():
            with ignore_path.open("r") as detignore_file:
                ignore.extend(detignore_file)
        ignore_spec = pathspec.PathSpec.from_lines(pathspec.patterns.GitWildMatchPattern, ignore)

        # We could use pathlib.Path.rglob for scanning the directory;
        # however, the Python documentation claims a warning that rglob may be
        # inefficient on large directory trees, so we use the older os.walk().
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

                self.add_v1File(v1File_from_local_dir(entry_path, dir_path))
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
                    entry = v1File_from_local_file(entry_path, file_path)
                except OSError:
                    print(f"Error reading '{entry_path}', skipping this file.")
                    continue

                self.add_v1File(entry)
                self.update_msg()

    def get_items(self) -> List[bindings.v1File]:
        print()

        # Check for conflicting root items, which may arise when --include conflicts with either
        # the --context or another --include.
        root_items = (f.path for f in self.items if "/" not in f.path.rstrip("/"))
        duplicates = [k for k, v in collections.Counter(root_items).items() if v > 1]
        if len(duplicates) == 1:
            raise ValueError(f"duplicate path detected: {repr(duplicates[0])}")
        elif duplicates:
            raise ValueError(f"duplicate paths detected: {duplicates}")

        return self.items


def read_v1_context(
    context_root: Optional[pathlib.Path],
    includes: Iterable[pathlib.Path] = (),
    limit: int = constants.MAX_CONTEXT_SIZE,
) -> List[bindings.v1File]:
    """
    Return a list of v1Files suitable for submitting a context directory over the v1 REST API.

    A .detignore file in a context or include directory, if specified, indicates the wildcard paths
    that should be ignored.  File paths inside the context_root are relative to the context_root.
    File paths from includes are prefixed by the basename of the included directory.
    """

    if context_root is None and not includes:
        return []

    builder = _Builder(limit)

    if context_root is not None:
        # --context must always be a directory.
        if context_root.is_file():
            raise ValueError(
                f"context path '{context_root}' must be a directory, maybe use an include instead?"
            )
        builder.add(context_root, pathlib.Path(""))
    for i in includes:
        # Some paths like "/" don't have a name we can assign within the tarball.
        name = i.resolve().name
        if not name:
            raise ValueError(f"unable to determine the name of include '{i}'")
        builder.add(i, pathlib.Path(name))

    return builder.get_items()


def read_legacy_context(
    context_root: Optional[pathlib.Path],
    includes: Iterable[pathlib.Path] = (),
    limit: int = constants.MAX_CONTEXT_SIZE,
) -> LegacyContext:
    return [v1File_to_dict(f) for f in read_v1_context(context_root, includes, limit)]
