import pathlib
from typing import List

from determined.common import context
from determined.common.api import bindings


def path_to_files(path: pathlib.Path) -> List[bindings.v1File]:
    files: List[bindings.v1File] = []
    for item in context.read_context(path)[0]:
        content = item["content"].decode("ascii")
        file = bindings.v1File(
            path=item["path"],
            type=item["type"],
            content=content,
            mtime=item["mtime"],
            uid=item["uid"],
            gid=item["gid"],
            mode=item["mode"],
        )
        files.append(file)
    return files
