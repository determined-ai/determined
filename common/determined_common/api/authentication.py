import getpass
import hashlib
import json
import os
import platform
import typing
from argparse import Namespace
from functools import wraps
from pathlib import Path
from typing import Any, Callable, Dict, NamedTuple, Optional, cast

from determined_common import api, constants
from determined_common.api import authentication as auth

Credentials = NamedTuple("Credentials", [("username", str), ("password", str)])

PASSWORD_SALT = "GubPEmmotfiK9TMD6Zdw"


def authentication_required(func: Callable[[Namespace], Any]) -> Callable[..., Any]:
    @wraps(func)
    def f(namespace: Namespace) -> Any:
        v = vars(namespace)
        auth.initialize_session(namespace.master, v.get("user"), try_reauth=True)
        return func(namespace)

    return f


def salt_and_hash(password: str) -> str:
    if password:
        return hashlib.sha512((PASSWORD_SALT + password).encode()).hexdigest()
    else:
        return password


class Session:
    def __init__(self, username: str, token: str):
        self.username = username
        self.token = token


class Authentication:
    _instance = None

    def __init__(self) -> None:
        self.token_store = TokenStore()
        self.session = None  # type: Optional[Session]

    @classmethod
    def instance(cls) -> "Authentication":
        if cls._instance is None:
            cls._instance = Authentication()
        return cls._instance

    def is_user_active(self, username: str) -> bool:
        return self.token_store.get_active_user() == username

    def get_session_user(self) -> str:
        """
        Returns the session user for the current session. If there is no active
        session, then an UnauthenticatedException will be raised.
        """
        if self.session is None:
            raise api.errors.UnauthenticatedException(username="")
        return self.session.username

    def get_session_token(self) -> str:
        """
        Returns the authentication token for the session user. If there is no
        active session, then an UnauthenticatedException will be raised.

        """
        if self.session is None:
            raise api.errors.UnauthenticatedException(username="")
        return self.session.token

    def reset_session(self) -> None:
        self.session = None


class TokenStore:
    def __init__(self) -> None:
        try:
            store = self._load_store()
        except api.errors.CorruptTokenCacheException as e:
            self.delete_token_cache()
            raise e
        self._tokens = store.get("tokens", {})
        self._active_user = typing.cast(Optional[str], store.get("active_user", None))

    @classmethod
    def delete_token_cache(cls) -> None:
        path = cls._get_token_cache_path()
        if path.exists():
            path.unlink()

    @staticmethod
    def _get_token_cache_path() -> Path:
        return get_config_path().joinpath("auth.json")

    def get_active_user(self) -> Optional[str]:
        """
        Gets the active user from the token cache.

        Returns: Optional[str] which is either the user if there is an active user or None
        otherwise.
        """
        return self._active_user

    @classmethod
    def _load_store(cls) -> Dict[str, Any]:
        path = cls._get_token_cache_path()
        if path.exists():
            with path.open() as fin:
                try:
                    store = typing.cast(Dict[str, Any], json.loads(fin.read()))

                    if not cls._validate_token_store(store):
                        raise api.errors.CorruptTokenCacheException

                    return store

                except json.JSONDecodeError:
                    raise api.errors.CorruptTokenCacheException
        else:
            return {}

    @staticmethod
    def _validate_token_store(store: Dict[str, Any]) -> bool:
        """
        _validate_token_store makes sure that the data in the token store makes sense
        (in the sense that the key/value types are what they should be).

        """
        if "active_user" in store:
            active_user = typing.cast(str, store["active_user"])
            if not isinstance(active_user, str):
                return False

        if "tokens" in store:
            tokens = typing.cast(Dict[str, str], store["tokens"])
            if not isinstance(tokens, dict):
                return False
            for k, v in tokens.items():
                if not isinstance(k, str):
                    return False
                if not isinstance(v, str):
                    return False
        return True

    @staticmethod
    def _create_det_path_if_necessary() -> None:
        path = get_config_path()
        if not path.exists():
            path.mkdir(parents=True, mode=0o700)

    def get_token(self, user: str) -> Optional[str]:
        if user in self._tokens:
            return typing.cast(str, self._tokens[user])
        return None

    def _write_store(self) -> None:
        self._create_det_path_if_necessary()
        cache_path = self._get_token_cache_path()
        store = {}
        if self._tokens is not None and len(self._tokens):
            store["tokens"] = self._tokens

        if self._active_user is not None:
            store["active_user"] = self._active_user

        with cache_path.open("w") as file_out:
            json.dump(store, file_out, indent=4, sort_keys=True)

    def drop_user(self, username: str) -> None:
        if username not in self._tokens:
            raise api.errors.UnauthenticatedException(username=username)

        del self._tokens[username]
        if self.get_active_user() == username:
            self._active_user = None
        self._write_store()

    def set_token(self, username: str, token: str) -> None:
        self._tokens[username] = token
        self._write_store()

    def set_active(self, username: str, active: bool) -> None:
        if username not in self._tokens:
            raise api.errors.UnauthenticatedException(username=username)

        self._active_user = username if active else None
        self._write_store()


def get_config_path() -> Path:
    system = platform.system()
    if "Linux" in system and "XDG_CONFIG_HOME" in os.environ:
        config_path = Path(os.environ["XDG_CONFIG_HOME"])
    elif "Darwin" in system:
        config_path = Path.home().joinpath("Library").joinpath("Application Support")
    elif "Windows" in system and "LOCALAPPDATA" in os.environ:
        config_path = Path(os.environ["LOCALAPPDATA"])
    else:
        config_path = Path.home().joinpath(".config")

    return config_path.joinpath("determined")


def initialize_session(
    master_address: str, requested_user: Optional[str] = None, try_reauth: bool = False
) -> None:
    auth = Authentication.instance()

    session_user = (
        requested_user or auth.token_store.get_active_user() or constants.DEFAULT_DETERMINED_USER
    )

    token = auth.token_store.get_token(session_user)
    if token is not None and not _is_token_valid(master_address, token):
        auth.token_store.drop_user(session_user)
        token = None

    if token is not None:
        auth.session = api.Session(session_user, token)
        return

    if token is None and not try_reauth:
        raise api.errors.UnauthenticatedException(username=session_user)

    password = None
    if session_user == constants.DEFAULT_DETERMINED_USER:
        password = constants.DEFAULT_DETERMINED_PASSWORD
    elif session_user is None:
        session_user = input("Username: ")

    if password is None:
        password = api.salt_and_hash(
            getpass.getpass("Password for user '{}': ".format(session_user))
        )

    token = do_login(master_address, auth, session_user, password)

    auth.token_store.set_token(session_user, token)

    # If the user wasn't set with the '-u' option and the session_user
    # is the default user, tag them as being the active user.
    if requested_user is None and session_user == constants.DEFAULT_DETERMINED_USER:
        auth.token_store.set_active(session_user, True)

    auth.session = api.Session(session_user, token)


def _is_token_valid(master_address: str, token: str) -> bool:
    """
    Find out whether the given token is valid by attempting to use it
    on the "/users/me" endpoint.
    """
    headers = {"Authorization": "Bearer {}".format(token)}
    try:
        r = api.get(master_address, "users/me", headers=headers, authenticated=False)
    except (api.errors.UnauthenticatedException, api.errors.APIException):
        return False

    return r.status_code == 200


def do_login(master_address: str, auth: Authentication, username: str, password: str) -> str:
    r = api.post(
        master_address,
        "login",
        body={"username": username, "password": password},
        authenticated=False,
    )

    token = cast(str, r.json()["token"])

    auth.token_store.set_token(username, token)

    return token
