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

from determined.common.api import authentication


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


class _BrokenCertStore(Exception):
    pass


def _load_cert_store(path: pathlib.Path) -> Dict[str, str]:
    if not path.exists():
        return {}
    try:

        with path.open() as f:
            content = f.read()

        try:
            store = json.loads(content)
        except json.JSONDecodeError:
            raise _BrokenCertStore()

            if not isinstance(store, dict):
                raise _BrokenCertStore()

        # Store must be a dictionary.
        if not isinstance(store, dict):
            raise _BrokenCertStore()

        # All keys are url's, all values are pem-encoded certs.
        for k, v in store.items():
            if not isinstance(k, str):
                raise _BrokenCertStore()
            if not isinstance(v, str):
                raise _BrokenCertStore()

    except _BrokenCertStore:
        path.unlink()
        return {}

    return cast(Dict[str, str], store)


@contextlib.contextmanager
def _modifiable_store(path: pathlib.Path) -> Iterator[Dict["str", "str"]]:
    """
    Yields the appropriate store that can be modified, and the modified result will be written
    back to file.

    The modified store is also saved to the local object.
    """
    path = path
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    # Decide on paths for a lock file and a temp files (during writing).
    temp = pathlib.Path(str(path) + ".temp")
    lock = pathlib.Path(str(path) + ".lock")

    with filelock.FileLock(lock):
        store = _load_cert_store(path)

        # No need for try/finally, because we don't update the file after failures.
        yield store

        with temp.open("w") as f:
            json.dump(store, f, indent=4, sort_keys=True)
        temp.rename(path)


def get_cert(path: pathlib.Path, url: str) -> Optional[str]:
    if not path.exists():
        return None
    # Technically this doesn't have to be modfiable, but it is unlikely to matter.
    with _modifiable_store(path) as store:
        return store.get(url)


def set_cert(path: pathlib.Path, url: str, cert_pem: str) -> None:
    with _modifiable_store(path) as store:
        store[url] = cert_pem


def delete_cert(path: pathlib.Path, url: str) -> None:
    with _modifiable_store(path) as store:
        if url in store:
            del store[url]


def maybe_shim_old_cert_store(
    old_path: pathlib.Path, new_path: pathlib.Path, master_url: str
) -> None:
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
    return authentication.get_config_path().joinpath("certs.json")


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
        old_path = authentication.get_config_path().joinpath("master.crt")
        maybe_shim_old_cert_store(old_path, store_path, master_url)
        cert_pem = get_cert(store_path, master_url)

    env_name = os.environ.get("DET_MASTER_CERT_NAME")
    if env_name == "":
        env_name = None

    return Cert(cert_pem=cert_pem, noverify=noverify, name=env_name)
