import base64
import collections
import os
import pathlib
import tarfile
from typing import Any, Dict, List, Optional, Tuple

import pathspec

from determined.common import check, constants
from determined.common.util import sizeof_fmt


class ContextItem:
    """
    ContextItem wraps the content and metadata of a file or a directory.
    """

    def __init__(self, path: str):
        self.path = path
        self.type = ord(tarfile.REGTYPE)
        self.uid = 0
        self.gid = 0
        self.content = bytes()
        self.mtime = -1
        self.mode = -1

    @property
    def size(self) -> int:
        if self.content:
            return len(self.content)
        return 0

    def dict(self) -> Dict[str, Any]:
        d = {"path": self.path, "type": self.type, "uid": self.uid, "gid": self.gid}
        if self.type in (ord(tarfile.REGTYPE), ord(tarfile.DIRTYPE)):
            d["content"] = self.content
        if self.mtime != -1:
            d["mtime"] = self.mtime
        if self.mode != -1:
            d["mode"] = self.mode
        return d

    @classmethod
    def from_content_str(cls, path: str, content: str) -> "ContextItem":
        context_item = ContextItem(path)
        context_item.type = ord(tarfile.REGTYPE)
        context_item.content = base64.b64encode(content.encode("utf-8"))
        return context_item

    @classmethod
    def from_local_file(cls, path: str, local_path: pathlib.Path) -> "ContextItem":
        context_item = ContextItem(path)
        context_item.type = ord(tarfile.REGTYPE)
        context_item.mtime = int(local_path.stat().st_mtime)
        context_item.mode = local_path.stat().st_mode
        with local_path.open("rb") as f:
            content = f.read()
            context_item.content = base64.b64encode(content)
        return context_item

    @classmethod
    def from_local_dir(cls, path: str, local_path: pathlib.Path) -> "ContextItem":
        context_item = ContextItem(path)
        context_item.type = ord(tarfile.DIRTYPE)
        context_item.mtime = int(local_path.stat().st_mtime)
        context_item.mode = local_path.stat().st_mode
        return context_item


class Context:
    """
    Context wraps the content and metadata of a collection of files and directories.
    """

    def __init__(self) -> None:
        self._items = {}  # type: Dict[str, ContextItem]
        self._size = 0

    def __len__(self) -> int:
        return len(self._items)

    @property
    def size(self) -> int:
        return self._size

    @property
    def entries(self) -> collections.abc.ValuesView:
        return self._items.values()

    def add_item(self, entry: ContextItem) -> None:
        self._items[entry.path] = entry
        self._size += entry.size

    @classmethod
    def from_local(
        cls,
        local_path: pathlib.Path,
        limit: int = constants.MAX_CONTEXT_SIZE,
    ) -> "Context":
        """
        Given the path to a local directory, return a Context object.

        A .detignore file in the directory, if specified, indicates the wildcard paths
        that should be ignored. File paths are represented as relative paths (relative to
        the root directory).
        """

        context = Context()
        local_path = local_path.resolve()

        if not local_path.exists():
            raise Exception("Path '{}' doesn't exist".format(local_path))

        if local_path.is_file():
            raise ValueError("Path '{}' must be a directory".format(local_path))

        root_path = local_path

        ignore = list(constants.DEFAULT_DETIGNORE)
        ignore_path = root_path.joinpath(".detignore")
        if ignore_path.is_file():
            with ignore_path.open("r") as detignore_file:
                ignore.extend(detignore_file)
        ignore_spec = pathspec.PathSpec.from_lines(pathspec.patterns.GitWildMatchPattern, ignore)

        msg = "Preparing files (in {}) to send to master... {} and {} files".format(
            root_path, sizeof_fmt(0), 0
        )
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

                context.add_item(ContextItem.from_local_dir(entry_path, dir_path))

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
                    entry = ContextItem.from_local_file(entry_path, file_path)
                except OSError:
                    print("Error reading '{}', skipping this file.".format(entry_path))
                    continue

                context.add_item(entry)
                if context.size > limit:
                    print()
                    raise ValueError(
                        "Directory '{}' exceeds the maximum allowed size {}.\n"
                        "Consider using a .detignore file to specify that certain files "
                        "or directories should be omitted from the model.".format(
                            root_path, sizeof_fmt(constants.MAX_CONTEXT_SIZE)
                        )
                    )

                print(" " * len(msg), end="\r")
                msg = "Preparing files ({}) to send to master... {} and {} files".format(
                    root_path, sizeof_fmt(context.size), len(context)
                )
                print(msg, end="\r", flush=True)
        print()
        return context


def read_context(
    local_path: pathlib.Path,
    limit: int = constants.MAX_CONTEXT_SIZE,
) -> Tuple[List[Dict[str, Any]], int]:
    context = Context.from_local(local_path, limit)
    return [e.dict() for e in context.entries], context.size


def read_single_file(file_path: Optional[pathlib.Path]) -> Tuple[bytes, int]:
    """
    Given a path to a file, return the base64-encoded contents of the file and its original size.
    """
    if not file_path:
        return b"", 0

    check.check_true(file_path.is_file(), 'The file at "{}" could not be found'.format(file_path))

    content = file_path.read_bytes()

    return base64.b64encode(content), len(content)


def get_invalid_model_def_path_message() -> str:
    """
    get_invalid_model_def_path_message is used by get_all_file_entries to generate an appropriate
    error message to display when a user tries to create an experiment using a model definition
    contained in a directory whose name conflicts with the Determined package name
    (i.e. "determined").

    The function is also used in test_cli.py::test_create_reject_bad_path.
    """
    return 'A model definition cannot be contained in a directory named "determined"'
