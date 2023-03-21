import contextlib
import logging
import os
import pathlib
import shutil
from typing import Any, Callable, Dict, Iterator, List, Optional, Union

from determined import errors
from determined.common import check, storage

# Based on shutil.copytree and shutil._copytree (for Python 3.8). Compared to the original
# implementation this code delays creating new directories during traversal, such that dir
# is created only when (1) selector(dir)==True or (2) selector(dir/subdir/path)==True.
# The code is simplified to rely on default values for:
# symlinks=False,
# ignore=None,
# copy_function=shutil.copy2,
# ignore_dangling_symlinks=False,
# dirs_exist_ok = True.


def _copytree(
    entries: List,
    src: str,
    dst: str,
    selector: Optional[Callable[[str], bool]],
    src_root: str,
) -> str:
    errors = []
    have_copied = False
    for srcobj in entries:
        srcname = os.path.join(src, srcobj.name)
        dstname = os.path.join(dst, srcobj.name)
        src_relpath = os.path.relpath(srcname, src_root)
        try:
            if srcobj.is_dir():
                # Directories are created here only if they are specified
                # in the selector. If a directory is not specified in
                # the selector, and it is required by any nested files,
                # the directory will be created then.
                if selector is None or selector(src_relpath + "/"):
                    os.makedirs(dstname, exist_ok=True)
                    have_copied = True
                copytree(
                    srcobj,
                    dstname,
                    selector,
                    src_root,
                )
            else:
                # If selector is None all files are copied; if selector is not None
                # then files are copied according to the selector. Before files
                # are copied all top directory structure is created. This ensures
                # that copied dirs are not dangling.
                if selector is None or selector(src_relpath):
                    have_copied = True
                    os.makedirs(dst, exist_ok=True)
                    shutil.copy2(srcobj, dstname)
        # catch the Error from the recursive copytree so that we can
        # continue with other files
        except shutil.Error as err:
            errors.extend(err.args[0])
        except OSError as why:
            errors.append((srcname, dstname, str(why)))
    if have_copied:
        try:
            shutil.copystat(src, dst)
        except OSError as why:
            # Copying file access times may fail on Windows
            if getattr(why, "winerror", None) is None:
                errors.append((src, dst, str(why)))
    if errors:
        raise shutil.Error(errors)
    return dst


def copytree(
    src: str,
    dst: str,
    selector: Optional[Callable[[str], bool]] = None,
    src_root: Optional[str] = None,
) -> str:
    if src_root is None:
        src_root = src
    with os.scandir(src) as itr:
        entries = list(itr)
    return _copytree(
        entries=entries,
        src=src,
        dst=dst,
        selector=selector,
        src_root=src_root,
    )


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

    def post_store_path(self, src: Union[str, os.PathLike], dst: str) -> None:
        """
        Nothing to clean up after writing directly to shared_fs.
        """
        pass

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
            logging.info(f"Storage directory does not exist: {storage_dir}")
            return
        if not os.path.isdir(storage_dir):
            raise errors.CheckpointNotFound(f"Storage path is not a directory: {storage_dir}")
        shutil.rmtree(storage_dir, ignore_errors=False)

    def upload(
        self, src: Union[str, os.PathLike], dst: str, paths: Optional[storage.Paths] = None
    ) -> None:
        src = os.fspath(src)

        if paths is None:
            selector = None
        else:

            def selector(x: str) -> bool:
                assert paths is not None
                return x in paths

        dst = os.path.join(self._base_path, dst)
        copytree(src, dst, selector=selector)

    def download(
        self,
        src: str,
        dst: Union[str, os.PathLike],
        selector: Optional[storage.Selector] = None,
    ) -> None:
        dst = os.fspath(dst)

        try:
            src = os.path.join(self._base_path, src)
            copytree(src, dst, selector=selector)
        except FileNotFoundError:
            raise errors.CheckpointNotFound(
                f"Did not find checkpoint {src} in shared_fs storage"
            ) from None
