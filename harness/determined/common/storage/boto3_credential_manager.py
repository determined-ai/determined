import logging
import os
import threading
import time
from typing import Any

import boto3
from botocore import session
from botocore.compat import ensure_unicode
from botocore.credentials import (
    CredentialProvider,
    Credentials,
    ReadOnlyCredentials,
    SharedCredentialProvider,
)
from botocore.session import get_session


class RefreshableCredentialProvider(CredentialProvider):  # type: ignore
    """
    Creates a refreshable credential provider class given an existing credential provider in
    the boto3 credential chain.

    """

    METHOD = "managed-refresh-cred"

    def __init__(self, credential_provider: SharedCredentialProvider, check_every: int = 2) -> None:
        super().__init__()
        self.check_every = check_every
        self.credential_provider = credential_provider

    def load(self) -> Credentials:
        return self.credential_provider.load() and RefreshableSharedCredentials(
            credentials_provider=self.credential_provider, check_every=self.check_every
        )


class RefreshableSharedCredentials(Credentials):  # type: ignore
    def __init__(
        self,
        check_every: int,
        credentials_provider: SharedCredentialProvider,
    ):
        self._credentials_provider = credentials_provider
        self._check_every = check_every
        self._lock = threading.Lock()
        self._check_time = time.time() + check_every
        self._load_and_set_credentials()

    def _load_and_set_credentials(self) -> None:
        credentials = self._credentials_provider.load()
        self._last_loaded = self._credentials_modified_time()
        self.access_key = credentials.access_key
        self.secret_key = credentials.secret_key
        self.token = credentials.token
        self._frozen_credentials = ReadOnlyCredentials(
            credentials.access_key, credentials.secret_key, credentials.token
        )

    def _credentials_file(self) -> Any:
        path = self._credentials_provider._creds_filename
        path = os.path.expandvars(path)
        path = os.path.expanduser(path)
        return path

    def _credentials_modified_time(self) -> float:
        credentials_file = self._credentials_file()
        return os.stat(credentials_file).st_mtime

    def _refresh_needed(self) -> bool:
        return self._credentials_modified_time() != self._last_loaded

    def _refresh(self) -> None:
        now = time.time()
        # Check before acquiring lock to prevent excessive locking
        if now < self._check_time:
            return
        with self._lock:
            # Real time check after acquiring lock
            if now < self._check_time:
                return
            self._check_time = now + self._check_every
            if self._refresh_needed():
                logging.info("credential file changes detected, refreshing credentials")
                self._load_and_set_credentials()

    def get_frozen_credentials(self) -> ReadOnlyCredentials:
        self._refresh()
        with self._lock:
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
    managed_credential_provider = RefreshableCredentialProvider(
        check_every=2, credential_provider=credential_provider
    )
    credential_resolver.insert_before(
        name=provider_name, credential_provider=managed_credential_provider
    )


def initialize_boto3_credential_providers() -> None:
    session = get_session()
    register_credential_provider(session, provider_name=SharedCredentialProvider.METHOD)
    boto3.setup_default_session(botocore_session=session)
