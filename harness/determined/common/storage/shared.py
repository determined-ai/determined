import contextlib
import logging
import os
import pathlib
import shutil
from typing import Any, Dict, Iterator, List, Optional, Union

from determined import errors
from determined.common import check, storage


def _full_storage_path(
    host_path: str,
    storage_path: Optional[str] = None,
    container_path: Optional[str] = None,
) -> str:
    """
    Return the full path to the storage_path, either as a subdirectory of the host_path in the
    host environment, where container_path must be None, or as a subdirectory of the container_path
    when in the container enviornment, where container_path must not be None.
    """
    check.true(os.path.isabs(host_path), "`host_path` must be an absolute path.")

    if storage_path is None:
        return host_path if container_path is None else container_path

    # Note that os.path.join() will just return storage_path when it is absolute.
    abs_path = os.path.normpath(os.path.join(host_path, storage_path))
    check.true(abs_path.startswith(host_path), "storage path must be a subdirectory of host path.")
    storage_path = os.path.relpath(abs_path, host_path)

    return os.path.join(host_path if container_path is None else container_path, storage_path)


class SharedFSStorageManager(storage.StorageManager):
    """
    Store and load storages from a shared file system. Each agent should
    have this shared file system mounted in the same location defined by the
    `host_path`.
    """

    @classmethod
    def from_config(
        cls, config: Dict[str, Any], container_path: Optional[str]
    ) -> "SharedFSStorageManager":
        allowed_keys = {"host_path", "storage_path", "container_path", "propagation"}
        for key in config.keys():
            check.is_in(key, allowed_keys, "extra key in shared_fs config")
        check.is_in("host_path", config, "shared_fs config is missing host_path")
        # Ignore legacy configuration values propagation and container_path.
        base_path = _full_storage_path(
            config["host_path"], config.get("storage_path"), container_path
        )
        return cls(base_path)

    def post_store_path(
        self, src: Union[str, os.PathLike], dst: str, paths: Optional[storage.Paths] = None
    ) -> None:
        """
        Nothing to clean up after writing directly to shared_fs.
        """
        if paths is not None:
            logging.warning(
                "Ignoring partial checkpoint upload to shared_fs; all files written were written "
                "directly to shared_fs."
            )

    def store_path_is_direct_access(self) -> bool:
        return True

    @contextlib.contextmanager
    def restore_path(
        self, src: str, selector: Optional[storage.Selector] = None
    ) -> Iterator[pathlib.Path]:
        """
        Prepare a local directory exposing the checkpoint. Do some simple checks to make sure the
        configuration seems reasonable.
        """
        if selector is not None:
            logging.warning(
                "Ignoring partial checkpoint download from shared_fs; all files will be directly "
                "accessible from shared_fs."
            )
        check.true(
            os.path.exists(self._base_path),
            f"Storage directory does not exist: {self._base_path}. Please verify that you are "
            "using the correct configuration value for checkpoint_storage.host_path",
        )
        storage_dir = os.path.join(self._base_path, src)
        if not os.path.exists(storage_dir):
            raise errors.CheckpointNotFound(f"Did not find checkpoint {src} in shared_fs storage")
        yield pathlib.Path(storage_dir)

    def delete(self, tgt: str) -> None:
        """
        Delete the stored data from persistent storage.
        """
        storage_dir = os.path.join(self._base_path, tgt)

        if not os.path.exists(storage_dir):
            raise errors.CheckpointNotFound(f"Storage directory does not exist: {storage_dir}")
        if not os.path.isdir(storage_dir):
            raise errors.CheckpointNotFound(f"Storage path is not a directory: {storage_dir}")
        shutil.rmtree(storage_dir, ignore_errors=False)

    def upload(
        self, src: Union[str, os.PathLike], dst: str, paths: Optional[storage.Paths] = None
    ) -> None:
        src = os.fspath(src)

        if paths is None:
            ignore = None
        else:
            paths_set = set(paths)

            def ignore(ign_dir: str, names: List[str]) -> List[str]:
                out = []
                # rel_dir would be "subdir" instead of "/determined_shared_fs/UUID/subdir"
                rel_dir = os.path.relpath(ign_dir, src)
                for name in names:
                    # ckp_path would be "subdir/file"
                    ckpt_path = os.path.join(rel_dir, name)
                    if ckpt_path not in paths_set:
                        out.append(name)
                return out

        shutil.copytree(src, os.path.join(self._base_path, dst), ignore=ignore)

    def download(
        self,
        src: str,
        dst: Union[str, os.PathLike],
        selector: Optional[storage.Selector] = None,
    ) -> None:
        dst = os.fspath(dst)

        maybe_dangling = []

        if selector is None:
            ignore = None
        else:

            def ignore(ign_dir: str, names: List[str]) -> List[str]:
                out: List[str] = []
                if selector is None:
                    return out
                # rel_dir would be "subdir" instead of "/determined_shared_fs/UUID/subdir"
                rel_dir = os.path.relpath(ign_dir, src)
                for name in names:
                    # ckpt_path would be "subdir/file"
                    ckpt_path = os.path.join(rel_dir, name)
                    if selector(ckpt_path):
                        # The user wants this file or directory.
                        continue
                    # src_path would be "/determined_shared_fs/UUID/subdir/file"
                    src_path = os.path.join(ign_dir, name)
                    if os.path.isdir(src_path):
                        # The user does not want this directory, but we don't yet know if there
                        # might be a subfile or subdirectory which they do want.  Let copytree
                        # continue and revisit this later.
                        maybe_dangling.append(ckpt_path)
                        continue
                    out.append(name)
                return out

        try:
            shutil.copytree(os.path.join(self._base_path, src), dst, ignore=ignore)
        except FileNotFoundError:
            raise errors.CheckpointNotFound(
                f"Did not find checkpoint {src} in shared_fs storage"
            ) from None

        # Any directory which was not wanted (but which we had to recurse into anyway), we now
        # attempt to remove.  By traversing the list in reverse order, any terminal maybe_dangling
        # directories will be removed, and the others will raise OSErrors, which we can ignore.
        for dangling in reversed(maybe_dangling):
            try:
                os.rmdir(os.path.join(dst, dangling))
            except OSError:
                pass
