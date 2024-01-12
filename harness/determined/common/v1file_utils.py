import base64
import pathlib
import tarfile
from typing import Any, Dict

from determined.common.api import bindings


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
