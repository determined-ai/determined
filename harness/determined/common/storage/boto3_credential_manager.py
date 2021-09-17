import logging
import os
import threading
from datetime import timedelta
from typing import Any, Callable, Optional

import boto3
from botocore import session
from botocore.compat import ensure_unicode
from botocore.credentials import (
    CredentialProvider,
    Credentials,
    ReadOnlyCredentials,
    SharedCredentialProvider,
    _local_now,
)
from botocore.session import get_session


class RefreshedCredentialProvider(CredentialProvider):  # type: ignore
    """
    Creates a refreshable credential provider class given an existing credential provider in
    the boto3 credential chain.

    """

    METHOD = "managed-refresh-cred"

    def __init__(self, credential_provider: SharedCredentialProvider, check_every: int = 2) -> None:
        super().__init__()
        self.check_every = timedelta(minutes=check_every)
        self.credential_provider = credential_provider
        self.last_loaded = self._credentials_modified_time()

    def load(self) -> Optional[Credentials]:
        credentials = self._load_credentials()
        return credentials and RefreshedSharedCredentials(
            refresh_using=self._load_credentials,
            refresh_needed=self._reload_needed,
            check_every=self.check_every,
            access_key=credentials.access_key,
            secret_key=credentials.secret_key,
            token=credentials.token,
        )

    def _reload_needed(self) -> bool:
        return self._credentials_modified_time() > self.last_loaded

    def _load_credentials(self) -> Credentials:
        self.last_loaded = self._credentials_modified_time()
        return self.credential_provider.load()

    def _credentials_file(self) -> Any:
        path = self.credential_provider._creds_filename
        path = os.path.expandvars(path)
        path = os.path.expanduser(path)
        return path

    def _credentials_modified_time(self) -> float:
        credentials_file = self._credentials_file()
        return os.stat(credentials_file).st_mtime


class RefreshedSharedCredentials(Credentials):  # type: ignore
    def __init__(
        self,
        access_key: str,
        secret_key: str,
        token: str,
        check_every: timedelta,
        refresh_using: Callable,
        refresh_needed: Callable,
    ):
        self._refresh_using = refresh_using
        self._refresh_needed = refresh_needed
        self._access_key = ensure_unicode(access_key)
        self._secret_key = ensure_unicode(secret_key)
        self._token = token
        self._check_every = check_every
        self._lock = threading.Lock()
        self._check_time = self._next_check_time()
        self._frozen_credentials = ReadOnlyCredentials(access_key, secret_key, token)

    def _refresh(self) -> None:
        now = _local_now()
        if now < self._check_time:
            return
        with self._lock:
            if now < self._check_time:
                return
            self._check_time = self._next_check_time()
            if self._refresh_needed():
                logging.info("credential file changes detected, refreshing credentials")
                credentials = self._refresh_using()
                self.access_key = credentials.access_key
                self.secret_key = credentials.secret_key
                self.token = credentials.token

    def _next_check_time(self) -> Any:
        return _local_now() + self._check_every

    def get_frozen_credentials(self) -> ReadOnlyCredentials:
        self._refresh()
        return ReadOnlyCredentials(self._access_key, self._secret_key, self._token)

    @property
    def access_key(self) -> Any:
        self._refresh()
        return self._access_key

    @access_key.setter
    def access_key(self, value: str) -> None:
        self._access_key = ensure_unicode(value)

    @property
    def secret_key(self) -> Any:
        self._refresh()
        return self._secret_key

    @secret_key.setter
    def secret_key(self, value: str) -> None:
        self._secret_key = ensure_unicode(value)

    @property
    def token(self) -> str:
        self._refresh()
        return self._token

    @token.setter
    def token(self, value: str) -> Any:
        self._token = value


def register_credential_provider(session: session, provider_name: str) -> None:
    credential_resolver = session.get_component("credential_provider")
    credential_provider = credential_resolver.get_provider(provider_name)
    managed_credential_provider = RefreshedCredentialProvider(
        credential_provider=credential_provider
    )
    credential_resolver.insert_before(
        name=provider_name, credential_provider=managed_credential_provider
    )


def initialize_boto3_credential_providers() -> None:
    session = get_session()
    register_credential_provider(session, provider_name=SharedCredentialProvider.METHOD)
    boto3.setup_default_session(botocore_session=session)
