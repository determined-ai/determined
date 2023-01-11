import contextlib
import logging
import os
import pathlib
import shutil
import sys
import typing
from typing import Any, Dict, Iterator, List, Optional, Union

from determined import errors
from determined.common import check, storage

python_version = sys.version_info
if python_version.major == 3 and python_version.minor <= 7:
    # Copied from shutil (Python 3.8) to support copytree(dirs_exist_ok) function.
    # Should be dropped when support for Python 3.7 is removed.
    # BEGIN VENDORED CODE FROM SHUTIL
    import stat

    @typing.no_type_check
    def _copytree(
        entries,
        src,
        dst,
        symlinks,
        ignore,
        copy_function,
        ignore_dangling_symlinks,
        dirs_exist_ok=False,
    ):
        if ignore is not None:
            ignored_names = ignore(os.fspath(src), [x.name for x in entries])
        else:
            ignored_names = set()

        os.makedirs(dst, exist_ok=dirs_exist_ok)
        errors = []
        use_srcentry = copy_function is shutil.copy2 or copy_function is shutil.copy

        for srcentry in entries:
            if srcentry.name in ignored_names:
                continue
            srcname = os.path.join(src, srcentry.name)
            dstname = os.path.join(dst, srcentry.name)
            srcobj = srcentry if use_srcentry else srcname
            try:
                is_symlink = srcentry.is_symlink()
                if is_symlink and os.name == "nt":
                    # Special check for directory junctions, which appear as
                    # symlinks but we want to recurse.
                    lstat = srcentry.stat(follow_symlinks=False)
                    if lstat.st_reparse_tag == stat.IO_REPARSE_TAG_MOUNT_POINT:
                        is_symlink = False
                if is_symlink:
                    linkto = os.readlink(srcname)
                    if symlinks:
                        # We can't just leave it to `copy_function` because legacy
                        # code with a custom `copy_function` may rely on copytree
                        # doing the right thing.
                        os.symlink(linkto, dstname)
                        shutil.copystat(srcobj, dstname, follow_symlinks=not symlinks)
                    else:
                        # ignore dangling symlink if the flag is on
                        if not os.path.exists(linkto) and ignore_dangling_symlinks:
                            continue
                        # otherwise let the copy occur. copy2 will raise an error
                        if srcentry.is_dir():
                            copytree(
                                srcobj,
                                dstname,
                                symlinks,
                                ignore,
                                copy_function,
                                dirs_exist_ok=dirs_exist_ok,
                            )
                        else:
                            copy_function(srcobj, dstname)
                elif srcentry.is_dir():
                    copytree(
                        srcobj,
                        dstname,
                        symlinks,
                        ignore,
                        copy_function,
                        dirs_exist_ok=dirs_exist_ok,
                    )
                else:
                    # Will raise a SpecialFileError for unsupported file types
                    copy_function(srcobj, dstname)
            # catch the Error from the recursive copytree so that we can
            # continue with other files
            except shutil.Error as err:
                errors.extend(err.args[0])
            except OSError as why:
                errors.append((srcname, dstname, str(why)))
        try:
            shutil.copystat(src, dst)
        except OSError as why:
            # Copying file access times may fail on Windows
            if getattr(why, "winerror", None) is None:
                errors.append((src, dst, str(why)))
        if errors:
            raise shutil.Error(errors)
        return dst

    @typing.no_type_check
    def copytree(
        src,
        dst,
        symlinks=False,
        ignore=None,
        copy_function=shutil.copy2,
        ignore_dangling_symlinks=False,
        dirs_exist_ok=False,
    ):

        with os.scandir(src) as itr:
            entries = list(itr)
        return _copytree(
            entries=entries,
            src=src,
            dst=dst,
            symlinks=symlinks,
            ignore=ignore,
            copy_function=copy_function,
            ignore_dangling_symlinks=ignore_dangling_symlinks,
            dirs_exist_ok=dirs_exist_ok,
        )

    # END VENDORED CODE FROM SHUTIL

else:
    copytree = shutil.copytree


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
            raise errors.CheckpointNotFound(f"Storage directory does not exist: {storage_dir}")
        if not os.path.isdir(storage_dir):
            raise errors.CheckpointNotFound(f"Storage path is not a directory: {storage_dir}")
        shutil.rmtree(storage_dir, ignore_errors=False)

    def upload(self, src: Union[str, os.PathLike], dst: str) -> None:
        src = os.fspath(src)
        copytree(src, os.path.join(self._base_path, dst), dirs_exist_ok=True)

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
                assert selector
                out: List[str] = []

                start_path = os.path.join(self._base_path, src)

                # rel_dir would be "subdir" instead of "/determined_shared_fs/UUID/subdir"
                rel_dir = os.path.relpath(ign_dir, start_path)

                # rel_dir == "." happens only for the top dir.
                # Since users provide selector with respect to the current dir (w/o using "."),
                # let's convert "." to empty path.
                if rel_dir == ".":
                    rel_dir = ""

                for name in names:
                    # ckpt_path would be "subdir/file"
                    path = os.path.join(rel_dir, name)

                    # src_path would be "/determined_shared_fs/UUID/subdir/file"
                    src_path = os.path.join(ign_dir, name)
                    if os.path.isdir(src_path):
                        # shutil removes '/' from dir names; we will add it manually.
                        path = os.path.join(path, "")

                    if selector(path):
                        # The user wants this file or directory.
                        continue

                    if os.path.isdir(src_path):
                        # The user does not want this directory, but we don't yet know if there
                        # might be a subfile or subdirectory which they do want.  Let copytree
                        # continue and revisit this later.
                        maybe_dangling.append(path)
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
