import base64
import os
import pathlib
import tarfile
from typing import Any, Dict, List

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


def v1File_from_local_file(path: str, local_path: pathlib.Path) -> bindings.v1File:
    with local_path.open("rb") as f:
        content = base64.b64encode(f.read()).decode("utf8")
    st = local_path.stat()
    return bindings.v1File(
        path=path,
        type=ord(tarfile.REGTYPE),
        content=content,
        # Protobuf expects string-encoded int64
        mtime=str(int(st.st_mtime)),
        mode=st.st_mode,
        uid=0,
        gid=0,
    )


def v1File_from_local_dir(path: str, local_path: pathlib.Path) -> bindings.v1File:
    st = local_path.stat()
    return bindings.v1File(
        path=path,
        type=ord(tarfile.DIRTYPE),
        content="",
        # Protobuf expects string-encoded int64
        mtime=str(int(st.st_mtime)),
        mode=st.st_mode,
        uid=0,
        gid=0,
    )


def read_v1_context(
    local_path: pathlib.Path,
    limit: int = constants.MAX_CONTEXT_SIZE,
) -> List[bindings.v1File]:
    """
    Return a list of v1Files suitable for submitting a context directory over the v1 REST API.

    A .detignore file in the directory, if specified, indicates the wildcard paths
    that should be ignored. File paths are represented as relative paths (relative to
    the root directory).
    """

    items = []
    size = 0

    def add_item(f: bindings.v1File) -> None:
        nonlocal size
        items.append(f)
        size += v1File_size(f)

    local_path = local_path.resolve()

    if not local_path.exists():
        raise Exception(f"Path '{local_path}' doesn't exist")

    if local_path.is_file():
        raise ValueError(f"Path '{local_path}' must be a directory")

    root_path = local_path

    ignore = list(constants.DEFAULT_DETIGNORE)
    ignore_path = root_path.joinpath(".detignore")
    if ignore_path.is_file():
        with ignore_path.open("r") as detignore_file:
            ignore.extend(detignore_file)
    ignore_spec = pathspec.PathSpec.from_lines(pathspec.patterns.GitWildMatchPattern, ignore)

    msg = f"Preparing files (in {root_path}) to send to master... {sizeof_fmt(0)} and {0} files"
    print(msg, end="\r", flush=True)

    # We could use pathlib.Path.rglob for scanning the directory;
    # however, the Python documentation claims a warning that rglob may be
    # inefficient on large directory trees, so we use the older os.walk().
    for parent, dirs, files in os.walk(str(root_path)):
        for directory in dirs:
            dir_path = pathlib.Path(parent).joinpath(directory)
            dir_rel_path = dir_path.relative_to(root_path)

            # If the file matches any path specified in .detignore, then ignore it.
            if ignore_spec.match_file(str(dir_rel_path) + "/"):
                continue

            # Determined only supports POSIX-style file paths.  Use as_posix() in case this code
            # is executed in a non-POSIX environment.
            entry_path = dir_rel_path.as_posix()

            add_item(v1File_from_local_dir(entry_path, dir_path))

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
            entry_path = file_rel_path.as_posix()

            try:
                entry = v1File_from_local_file(entry_path, file_path)
            except OSError:
                print(f"Error reading '{entry_path}', skipping this file.")
                continue

            add_item(entry)
            if size > limit:
                print()
                raise ValueError(
                    f"Directory '{root_path}' exceeds the maximum allowed size "
                    f"{sizeof_fmt(constants.MAX_CONTEXT_SIZE)}.\n"
                    "Consider using a .detignore file to specify that certain files "
                    "or directories should be omitted from the context directory."
                )

            print(" " * len(msg), end="\r")
            msg = (
                f"Preparing files ({root_path}) to send to master... "
                f"{sizeof_fmt(size)} and {len(items)} files"
            )
            print(msg, end="\r", flush=True)
    print()
    return items


def read_legacy_context(
    local_path: pathlib.Path,
    limit: int = constants.MAX_CONTEXT_SIZE,
) -> LegacyContext:
    return [v1File_to_dict(f) for f in read_v1_context(local_path, limit)]
