import collections
import pathlib
from typing import Any, Dict, Iterable, List, Optional

from determined.common import constants, detignore, util, v1file_utils
from determined.common.api import bindings

LegacyContext = List[Dict[str, Any]]


class _Builder:
    def __init__(self, limit: int) -> None:
        self.limit = limit
        self.size = 0
        self.items = []  # type: List[bindings.v1File]
        self.msg = f"Preparing files to send to master... {util.sizeof_fmt(0)} and 0 files"
        print(self.msg, end="\r", flush=True)

    def add_v1File(self, f: bindings.v1File) -> None:
        self.items.append(f)
        self.size += v1file_utils.v1File_size(f)
        if self.size > self.limit:
            raise ValueError(
                "The total size of context directory and included files and directories exceeds "
                f" the maximum allowed size {util.sizeof_fmt(self.limit)}.\n"
                "Consider using either .detignore files inside directories to specify that certain "
                "files or subdirectories should be omitted."
            )

    def update_msg(self) -> None:
        print(" " * len(self.msg), end="\r")
        self.msg = (
            "Preparing files to send to master... "
            f"{util.sizeof_fmt(self.size)} and {len(self.items)} files"
        )
        print(self.msg, end="\r", flush=True)

    def add(self, root_path: pathlib.Path, entry_prefix: pathlib.Path) -> None:
        root_path = root_path.resolve()

        if not root_path.exists():
            raise ValueError(f"Path '{root_path}' doesn't exist")

        if root_path.is_file():
            self.add_v1File(v1file_utils.v1File_from_local_file(root_path.name, root_path))
            return

        if str(entry_prefix) != ".":
            # For non-context directories, include the root directory.
            self.add_v1File(v1file_utils.v1File_from_local_dir(str(entry_prefix), root_path))

        for file in detignore.os_walk_to_v1Files(root_path, entry_prefix):
            self.add_v1File(file)
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
    return [v1file_utils.v1File_to_dict(f) for f in read_v1_context(context_root, includes, limit)]
