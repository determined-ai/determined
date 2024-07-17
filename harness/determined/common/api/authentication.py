import contextlib
import getpass
import hashlib
import json
import os
import pathlib
import re
import sys
from typing import Any, Dict, Iterator, List, Optional, Tuple
from urllib import parse

import filelock

from determined.common import api, constants, util
from determined.common.api import bindings, certs

PASSWORD_SALT = "GubPEmmotfiK9TMD6Zdw"


def salt_and_hash(password: str) -> str:
    if password:
        return hashlib.sha512((PASSWORD_SALT + password).encode()).hexdigest()
    else:
        return password


def warn_about_complexity(e: ValueError) -> None:
    print(
        "Warning: your password does not appear to satisfy "
        + f"recommended complexity requirements:\n{e}\n"
        + "Please change your password as soon as possible.",
        file=sys.stderr,
    )


def check_password_complexity(password: Optional[str]) -> None:
    """Raises a ValueError if the password does not meet complexity requirements.

    The complexity requirements are:
        - Must be at least 8 characters long.
        - Must contain at least one upper-case letter.
        - Must contain at least one lower-case letter.
        - Must contain at least one number.

    Args:
        password: a password to check.

    Raises:
        ValueError: an error describing why the password does not meet complexity requirements.
    """
    # TODO: DET-10209 - this should either invoke a shared lib or call a server endpoint
    good = "\u2713 "  # ✓
    bad = "\u2717 "  # ✗

    results = []
    ok = True

    if not password:
        results.append(bad + "password cannot be blank")
        ok = False
    else:
        results.append(good + "password cannot be blank")

    if not password or len(password) < 8:
        results.append(bad + "password must have at least 8 characters")
        ok = False
    else:
        results.append(good + "password must have at least 8 characters")

    if not password or re.search(r"[A-Z]", password) is None:
        results.append(bad + "password must include an uppercase letter")
        ok = False
    else:
        results.append(good + "password must include an uppercase letter")

    if not password or re.search(r"[a-z]", password) is None:
        results.append(bad + "password must include a lowercase letter")
        ok = False
    else:
        results.append(good + "password must include a lowercase letter")

    if not password or re.search(r"\d", password) is None:
        results.append(bad + "password must include a number")
        ok = False
    else:
        results.append(good + "password must include a number")

    if not ok:
        raise ValueError("\n".join(results))


def get_det_username_from_env() -> Optional[str]:
    return os.environ.get("DET_USER")


def get_det_user_token_from_env() -> Optional[str]:
    return os.environ.get("DET_USER_TOKEN")


def get_det_password_from_env() -> Optional[str]:
    return os.environ.get("DET_PASS")


def login(
    master_address: str,
    username: str,
    password: str,
    cert: Optional[certs.Cert] = None,
) -> "api.Session":
    """
    Log in without considering or affecting the TokenStore on the file system.

    This sends a login request to the master in order to obtain a new token that can sign future
    requests to the master. This token is then baked into a new api.Session object for those future
    communications.

    Used as part of login_with_cache, and also useful in tests where you wish to not affect the
    TokenStore.

    Returns:
        A new, logged-in api.Session (one that has a valid token).
    """
    password = api.salt_and_hash(password)
    unauth_session = api.UnauthSession(master=master_address, cert=cert, max_retries=0)
    login = bindings.v1LoginRequest(username=username, password=password, isHashed=True)
    r = bindings.post_Login(session=unauth_session, body=login)
    return api.Session(master=master_address, username=username, token=r.token, cert=cert)


def default_load_user_password(
    requested_user: Optional[str],
    password: Optional[str],
    token_store: "TokenStore",
) -> Tuple[str, Optional[str], bool]:
    """
    Decide on a username and password for a login attempt.

    When values are explicitly provided, they should be honored.  But when they are not provided,
    check environment variables and the token store before falling back to the system default.

    Args:
        requested_user: a username explicitly provided by the end user
        password: a password explicitly provided by the end user

    Returns:
        A tuple of (username, Optional[password], was_fallback], where was_fallback indicates that
        we are returning the system default username and password.
    """
    # Always prefer an explicitly provided user/password.
    if requested_user:
        return requested_user, password, False

    # Next highest priority is user/password from environment.
    # Watch out! We have to check for DET_USER and DET_PASS, because containers will have DET_USER
    # set, but that doesn't overrule the active user in the TokenStore, because if the TokenStore in
    # the container has an active user, that means the user has explicitly ran `det user login`
    # inside the container.
    env_user = get_det_username_from_env()
    env_pass = get_det_password_from_env()
    if env_user is not None and env_pass is not None:
        return env_user, env_pass, False

    # Next priority is the active user in the token store.
    active_user = token_store.get_active_user()
    if active_user is not None:
        return active_user, password, False

    # Last priority is the default username and password.
    return (
        constants.DEFAULT_DETERMINED_USER,
        password or constants.DEFAULT_DETERMINED_PASSWORD,
        True,
    )


def login_with_cache(
    master_address: str,
    requested_user: Optional[str] = None,
    password: Optional[str] = None,
    cert: Optional[certs.Cert] = None,
) -> "api.Session":
    """
    Log in, preferring cached credentials in the TokenStore, if possible.

    This is the login path for nearly all user-facing cases.

    Unlike ``login``, this function may not send a login request to the master. It will instead
    first attempt to find a valid token in the TokenStore, and only if that fails will it post a
    login request to the master to generate a new one. As with ``login``, the token is then baked
    into a new ``api.Session`` object to sign future communication with master.

    There is also a special case for checking if the DET_USER_TOKEN is set in the environment (by
    the determined-master).  That must happen in this function because it is only used when no other
    login tokens are active, but it must be considered before asking the user for a password.

    As a somewhat surprising side-effect of re-using an existing token from the cache, it is
    actually possible in cache hit scenarios for an invalid password here to result in a valid login
    since the password is only used in a cache miss.

    Returns:
        A new, logged-in Session (one that has a valid token).
    """

    token_store = TokenStore(master_address)

    user, password, was_fallback = default_load_user_password(requested_user, password, token_store)

    # Check the token store if this session_user has a cached token. If so, check with the
    # master to verify it has not expired. Otherwise, let the token be None.
    token = token_store.get_token(user)
    if token is not None and not _is_token_valid(master_address, token, cert):
        token_store.drop_user(user)
        token = None

    if token is not None:
        return api.Session(master=master_address, username=user, token=token, cert=cert)

    # Special case: use token provided from the container environment if:
    # - No token was obtained from the token store already,
    # - There is a token available from the container environment, and
    # - No user was explicitly requested, or the requested user matches the token available in the
    #   container environment.
    if (
        get_det_username_from_env() is not None
        and get_det_user_token_from_env() is not None
        and requested_user in (None, get_det_username_from_env())
    ):
        env_user = get_det_username_from_env()
        assert env_user
        env_token = get_det_user_token_from_env()
        assert env_token
        return api.Session(master=master_address, username=env_user, token=env_token, cert=cert)

    if password is None:
        password = getpass.getpass(f"Password for user '{user}': ")

    try:
        sess = login(master_address, user, password, cert)
        user, token = sess.username, sess.token
    except api.errors.ForbiddenException:
        # Master will return a 403 if the user is not found, or if the password is incorrect.
        # This is the right response to a failed explicit login attempt. But in the "fallback" case,
        # a user hasn't provided login information. There, a 401 "maybe try logging in" is a more
        # appropriate response to login failure.
        if was_fallback:
            raise api.errors.UnauthenticatedException()
        raise

    try:
        check_password_complexity(password)
    except ValueError as e:
        warn_about_complexity(e)

    token_store.set_token(user, token)

    return sess


def logout(
    master_address: str,
    requested_user: Optional[str],
    cert: Optional[certs.Cert],
) -> Optional[str]:
    """
    Logout if there is an active session for this master/username pair, otherwise do nothing.

    A session is active when a valid token for it can be found. In that case, the token
    is sent to the master for invalidation and dropped from the token store.

    If requested_user is None, logout attempts to log out the token store's active_user.

    Logout does not affect the "active_user" entry itself in the token store,
    since the whole concept of an "active user" mostly belongs to the CLI and is
    handled explicitly by the CLI.

    Returns:
        The name of the user who was logged out, if any.
    """

    token_store = TokenStore(master_address)

    user, _, __ = default_load_user_password(requested_user, None, token_store)

    token = token_store.get_token(user)

    if token is None:
        return None

    token_store.drop_user(user)

    sess = api.Session(master=master_address, username=user, token=token, cert=cert)
    try:
        bindings.post_Logout(sess)
    except (api.errors.UnauthenticatedException, api.errors.APIException):
        # This session may have expired, but we don't care.
        pass

    return user


def logout_all(master_address: str, cert: Optional[certs.Cert]) -> None:
    token_store = TokenStore(master_address)

    users = token_store.get_all_users()

    for user in users:
        logout(master_address, user, cert)


def _is_token_valid(master_address: str, token: str, cert: Optional[certs.Cert]) -> bool:
    """
    Find out whether the given token is valid by attempting to use it
    on the "api/v1/me" endpoint.
    """
    sess = api.Session(master_address, username="ignored", token=token, cert=cert)
    try:
        r = sess.get("api/v1/me")
    except (api.errors.UnauthenticatedException, api.errors.APIException):
        return False

    return r.status_code == 200


class TokenStore:
    """
    TokenStore is a class for reading/updating a persistent store of user authentication tokens.
    TokenStore can remember tokens for many users for each of many masters.  It can also remembers
    one "active user" for each master, which is set via `det user login`.

    All updates to the file follow a read-modify-write pattern, and use file locks to protect the
    integrity of the underlying file cache.
    """

    def __init__(self, master_address: str, path: Optional[pathlib.Path] = None) -> None:
        if master_address != api.canonicalize_master_url(master_address):
            # This check is targeting developers of Determined, not users of Determined.
            raise RuntimeError(
                f"TokenStore created with non-canonicalized url: {master_address}; the master url "
                "should have been canonicalized as soon as it was received from the end-user."
            )

        self.master_address = master_address
        self.path = path or util.get_config_path().joinpath("auth.json")
        self.path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        # Decide on paths for a lock file and a temp files (during writing)
        self.temp = pathlib.Path(str(self.path) + ".temp")
        self.lock = str(self.path) + ".lock"

        with filelock.FileLock(self.lock):
            store = self._load_store_file()

        self._reconfigure_from_store(store)

    def _reconfigure_from_store(self, store: dict) -> None:
        substore = store.get("masters", {}).get(self.master_address, {})

        active_user = substore.get("active_user")
        assert isinstance(active_user, (str, type(None))), active_user
        self._active_user = active_user

        tokens = substore.get("tokens", {})
        assert isinstance(tokens, dict), tokens
        self._tokens = tokens

    def get_active_user(self) -> Optional[str]:
        return self._active_user

    def get_all_users(self) -> List[str]:
        return list(self._tokens)

    def get_token(self, user: str) -> Optional[str]:
        token = self._tokens.get(user)
        if token is not None:
            assert isinstance(token, str), "invalid cache; token must be a string"
        return token

    def delete_token_cache(self) -> None:
        with filelock.FileLock(self.lock):
            if self.path.exists():
                self.path.unlink()

    def drop_user(self, username: str) -> None:
        with self._persistent_store() as substore:
            tokens = substore.setdefault("tokens", {})
            if username in tokens:
                del tokens[username]

    def set_token(self, username: str, token: str) -> None:
        with self._persistent_store() as substore:
            tokens = substore.setdefault("tokens", {})
            tokens[username] = token

    def set_active(self, username: str) -> None:
        with self._persistent_store() as substore:
            tokens = substore.setdefault("tokens", {})
            if username not in tokens:
                raise api.errors.UnauthenticatedException()
            substore["active_user"] = username

    def clear_active(self) -> None:
        with self._persistent_store() as substore:
            substore.pop("active_user", None)

    @contextlib.contextmanager
    def _persistent_store(self) -> Iterator[Dict[str, Any]]:
        """
        Yields the appropriate store[self.master_address] that can be modified, and the modified
        result will be written back to file.

        Whatever updates are made will also be updated on self automatically.
        """
        with filelock.FileLock(self.lock):
            store = self._load_store_file()
            substore = store.setdefault("masters", {}).setdefault(self.master_address, {})

            # No need for try/finally, because we don't update the file after failures.
            yield substore

            # Reconfigure our cached variables.
            self._reconfigure_from_store(store)

            with self.temp.open("w") as f:
                json.dump(store, f, indent=4, sort_keys=True)
            self.temp.replace(self.path)

    def _load_store_file(self) -> Dict[str, Any]:
        """
        Read a token store from a file, shimming it to the most recent version if necessary.

        If a v0 store is found it will be reconfigured as a v1 store based on the master_address
        that is being currently requested.
        """
        try:
            if not self.path.exists():
                return {"version": 1}

            try:
                with self.path.open() as f:
                    store = json.load(f)
            except json.JSONDecodeError:
                raise api.errors.CorruptTokenCacheException()

            if not isinstance(store, dict):
                raise api.errors.CorruptTokenCacheException()

            version = store.get("version", 0)
            if version < 1:
                validate_token_store_v0(store)
                store = shim_store_v0(store, self.master_address)
            if version < 2:
                validate_token_store_v1(store)
                store = shim_store_v1(store)

            validate_token_store_v2(store)

            assert isinstance(store, dict), store
            return store

        except api.errors.CorruptTokenCacheException:
            # Delete invalid caches before exiting.
            self.path.unlink()
            raise


def shim_store_v0(v0: Dict[str, Any], master_address: str) -> Dict[str, Any]:
    """
    v1 schema is just a bit more nesting to support multiple masters.
    """
    v1 = {"version": 1, "masters": {master_address: v0}}
    return v1


def precanonicalize_v1_url(url: str) -> str:
    """
    Remove parts of the url that were ignored by old versions of determined but will be rejected by
    canonicalize_master_url() now.

    We want to use canonicalize_master_url() to ensure the proper canonical form but we must
    tolerate any urls which may previously have been written into the token_store.
    """
    # We need to prepend a scheme first, because urlparse() doesn't handle that case well.
    if url.startswith("https://"):
        default_port = 443
    elif url.startswith("http://"):
        default_port = 80
    else:
        url = f"http://{url}"
        default_port = 8080

    parsed = parse.urlparse(url)

    # Extract just hostname:port from the authority section of the url.
    port = parsed.port or default_port
    netloc = f"{parsed.hostname}:{port}"

    # Discard username, password, query, and fragment.
    return parse.urlunparse((parsed.scheme, netloc, parsed.path, "", "", "")).rstrip("/")


def shim_store_v1(v1: Dict[str, Any]) -> Dict[str, Any]:
    """
    v2 scheme is the same as v1 schema but with canonicalized master urls.
    """
    v1_masters = v1.get("masters", {})

    # Build a 1-to-many mapping of canonical master urls to v1 master urls.
    canonicals: Dict[str, List[Dict[str, Any]]] = {}
    for master_url, entry in v1_masters.items():
        try:
            precanonical_url = precanonicalize_v1_url(master_url)
            canonical_url = api.canonicalize_master_url(precanonical_url)
            canonicals.setdefault(canonical_url, []).append(entry)
        except ValueError:
            # Just in case precanonicalize_v1_url() didn't catch something, we drop it from
            # the authentication cache, instead of breaking the CLI completely.
            pass

    v2_masters: Dict[str, Any] = {}

    for canonical_url, entries in canonicals.items():
        # Keep one of every username/token pair.
        tokens = {}
        for entry in entries:
            for user, token in entry.get("tokens", {}).items():
                tokens[user] = token

        v2_entry: Dict[str, Any] = {"tokens": tokens}

        # Pick one active user, if there were any.
        active_users = [e["active_user"] for e in entries if "active_user" in e]
        if active_users:
            v2_entry["active_user"] = active_users[0]

        v2_masters[canonical_url] = v2_entry

    v2 = {"version": 2, "masters": v2_masters}
    return v2


def validate_one_master_entry(obj: Any) -> None:
    """A validation helper for various versioned validators."""

    if not isinstance(obj, dict):
        raise api.errors.CorruptTokenCacheException()

    if len(set(obj.keys()).difference({"active_user", "tokens"})) > 0:
        # Extra keys.
        raise api.errors.CorruptTokenCacheException()

    if "active_user" in obj:
        if not isinstance(obj["active_user"], str):
            raise api.errors.CorruptTokenCacheException()

    if "tokens" in obj:
        tokens = obj["tokens"]
        if not isinstance(tokens, dict):
            raise api.errors.CorruptTokenCacheException()
        for k, v in tokens.items():
            if not isinstance(k, str):
                raise api.errors.CorruptTokenCacheException()
            if not isinstance(v, str):
                raise api.errors.CorruptTokenCacheException()


def validate_dict_of_masters(masters: Any) -> None:
    """A validation helper for various versioned validators."""

    if not isinstance(masters, dict):
        raise api.errors.CorruptTokenCacheException()

    # Each entry of masters must be a master_url/substore pair.
    for key, val in masters.items():
        if not isinstance(key, str):
            raise api.errors.CorruptTokenCacheException()
        validate_one_master_entry(val)


def validate_token_store_v0(store: Any) -> None:
    """
    Valid v0 schema example:

        {
          "active_user": "user_a",
          "tokens": {
            "user_a": "TOKEN",
            "user_b": "TOKEN"
          }
        }
    """
    validate_one_master_entry(store)


def validate_token_store_v1(store: Any) -> None:
    """
    Valid v1 schema example:

        {
          "version": 1,
          "masters": {
            "master_url_a": {
              "active_user": "user_a",
              "tokens": {
                "user_a": "TOKEN",
                "user_b": "TOKEN"
              }
            },
            "master_url_b": {
              "active_user": "user_c",
              "tokens": {
                "user_c": "TOKEN",
                "user_d": "TOKEN"
              }
            }
          }
        }

    Note that store["masters"] is a mapping of string url's to valid v0 schemas.
    """
    if not isinstance(store, dict):
        raise api.errors.CorruptTokenCacheException()

    if set(store.keys()) > {"version", "masters"}:
        # Extra keys.
        raise api.errors.CorruptTokenCacheException()

    # Handle version.
    version = store.get("version")
    if version != 1:
        raise api.errors.CorruptTokenCacheException()

    if "masters" in store:
        validate_dict_of_masters(store["masters"])


def validate_token_store_v2(store: Any) -> None:
    """
    Valid v2 schema example:

        {
          "version": 2,
          "masters": {
            "master_url_a": {
              "active_user": "user_a",
              "tokens": {
                "user_a": "TOKEN",
                "user_b": "TOKEN"
              }
            },
            "master_url_b": {
              "active_user": "user_c",
              "tokens": {
                "user_c": "TOKEN",
                "user_d": "TOKEN"
              }
            }
          }
        }

    Note that store["masters"] is a mapping of string url's to valid v0 schemas.

    Note that v2 is the same as v1 except master url's must be canonicalized.
    """
    if not isinstance(store, dict):
        raise api.errors.CorruptTokenCacheException()

    if set(store.keys()) > {"version", "masters"}:
        # Extra keys.
        raise api.errors.CorruptTokenCacheException()

    # Handle version.
    version = store.get("version")
    if version != 2:
        raise api.errors.CorruptTokenCacheException()

    if "masters" in store:
        masters = store["masters"]
        validate_dict_of_masters(masters)

        if not all(api.canonicalize_master_url(key) == key for key in masters):
            # A non-canonical master url is present.
            raise api.errors.CorruptTokenCacheException()
