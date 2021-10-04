import atexit
import contextlib
import json
import logging
import os
import pathlib
import tempfile
from typing import Dict, Iterator, Optional, Union, cast

import certifi
import filelock

from determined.common import api, util


class Cert:
    def __init__(
        self,
        cert_pem: Optional[str] = None,
        noverify: bool = False,
        name: Optional[str] = None,
    ) -> None:
        if cert_pem is not None and noverify:
            raise AssertionError("you cannot set cert_pem with noverify=True")

        if name == "":
            name = None
        self._name = name

        if noverify:
            self._bundle = False  # type: Union[None, str, bool]
        elif cert_pem is None:
            self._bundle = None
        else:
            # Don't use NamedTemporaryFile, since it would make the file inaccessible by path on
            # Windows after this.
            # (see https://docs.python.org/3/library/tempfile.html#tempfile.NamedTemporaryFile)
            fd, combined_path = tempfile.mkstemp(prefix="det-master-cert-")
            atexit.register(os.unlink, combined_path)

            with open(fd, "wb") as out:
                with open(certifi.where(), "rb") as base_certs:
                    out.write(base_certs.read())
                out.write(b"\n")
                out.write(cert_pem.encode("utf8"))

            self._bundle = combined_path

    @property
    def bundle(self) -> Union[None, str, bool]:
        """
        The path to a file containing an SSL certificate to trust specifically for the master, if
        any, or False to disable cert verification entirely. If set to a path, it should always be a
        temporary file that we own and can delete.
        """
        return self._bundle

    @property
    def name(self) -> Optional[str]:
        """
        The name we use to verify the master certificate.
        """
        return self._name


cli_cert = None  # type: Optional[Cert]


class CertStore:
    """
    CertStore represents a persistent file-based record of certificates, each associated with a
    particular master url.
    """

    def __init__(self, path: pathlib.Path) -> None:
        self.path = path

    def _load_store_file(self) -> Dict[str, str]:
        if not self.path.exists():
            return {}

        try:
            try:
                with self.path.open() as f:
                    store = json.load(f)
            except json.JSONDecodeError:
                raise api.errors.CorruptCertificateCacheException()

            # Store must be a dictionary.
            if not isinstance(store, dict):
                raise api.errors.CorruptCertificateCacheException()

            # All keys are url's, all values are pem-encoded certs.
            for k, v in store.items():
                if not isinstance(k, str):
                    raise api.errors.CorruptCertificateCacheException()
                if not isinstance(v, str):
                    raise api.errors.CorruptCertificateCacheException()

            return cast(Dict[str, str], store)

        except api.errors.CorruptCertificateCacheException:
            self.path.unlink()
            raise

    @contextlib.contextmanager
    def _persistent_store(self) -> Iterator[Dict["str", "str"]]:
        """
        Yields a store that can be modified, and the modified result will be written back to file.
        """
        self.path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        # Decide on paths for a lock file and a temp files (during writing).
        temp = pathlib.Path(str(self.path) + ".temp")
        lock = str(self.path) + ".lock"

        with filelock.FileLock(lock):
            store = self._load_store_file()

            # No need for try/finally, because we don't update the file after failures.
            yield store

            with temp.open("w") as f:
                json.dump(store, f, indent=4, sort_keys=True)
            temp.replace(self.path)

    def get_cert(self, url: str) -> Optional[str]:
        """
        get_cert returns the contents of a cert (if any) that has been associated with the given
        url.
        """
        if not self.path.exists():
            return None
        # Technically this doesn't have to be modfiable, but it is unlikely to matter.
        with self._persistent_store() as store:
            return store.get(url)

    def set_cert(self, url: str, cert_pem: str) -> None:
        with self._persistent_store() as store:
            store[url] = cert_pem

    def delete_cert(self, url: str) -> None:
        with self._persistent_store() as store:
            if url in store:
                del store[url]


def maybe_shim_old_cert_store(
    old_path: pathlib.Path, new_path: pathlib.Path, master_url: str
) -> None:
    """
    maybe_shim_old_cert_store will detect when an old v0 cert store is present and will shim it to
    a v1 cert store.
    """
    if not old_path.exists():
        return None

    # Only try to shim when ONLY the old path exists.
    if not new_path.exists():
        with old_path.open("r") as f:
            pem_content = f.read()
        store = {master_url: pem_content}
        with new_path.open("w") as f:
            json.dump(store, f, indent=4, sort_keys=True)

    old_path.unlink()


def default_store() -> pathlib.Path:
    return util.get_config_path().joinpath("certs.json")


def default_load(
    master_url: str,
    explicit_path: Optional[str] = None,
    explicit_cert_name: Optional[str] = None,
    explicit_noverify: bool = False,
) -> Cert:
    """
    default_load takes as input the user-specified certificate-related configurations, reads
    environment variable configs, reads configs from the default store on the filesystem, and
    returns the resulting Cert object.

    Having all of the default loading logic in one function makes it easy to invoke the same logic
    in both the cli and the python sdk.
    """
    # Any explicit args causes us to ignore environment variables and defaults.
    if explicit_path or explicit_cert_name or explicit_noverify:
        if explicit_path:
            with open(explicit_path, "r") as f:
                cert_pem = f.read()  # type: Optional[str]
        else:
            cert_pem = None
        return Cert(cert_pem=cert_pem, noverify=explicit_noverify, name=explicit_cert_name)

    # Let any environment variable for CERT_FILE override the default store.
    env_path = os.environ.get("DET_MASTER_CERT_FILE")
    noverify = False
    cert_pem = None
    if env_path:
        if env_path.lower() == "noverify":
            noverify = True
        elif os.path.exists(env_path):
            with open(env_path, "r") as f:
                cert_pem = f.read()
        else:
            logging.warning(
                f"DET_MASTER_CERT_FILE={env_path} path not found; continuing without cert"
            )
    else:
        # Otherwise, look in the default location for cert_pem.
        store_path = default_store()
        cert_store = CertStore(path=store_path)
        old_path = util.get_config_path().joinpath("master.crt")
        maybe_shim_old_cert_store(old_path, store_path, master_url)
        cert_pem = cert_store.get_cert(master_url)

    env_name = os.environ.get("DET_MASTER_CERT_NAME")
    if env_name == "":
        env_name = None

    return Cert(cert_pem=cert_pem, noverify=noverify, name=env_name)
