import logging
from datetime import timedelta
from typing import Any, Dict, Optional

import boto3
from boto3 import Session
from botocore.credentials import (
    CredentialProvider,
    Credentials,
    RefreshableCredentials,
    SharedCredentialProvider,
    _local_now,
)
from botocore.session import get_session


class Boto3Manager:
    """
    Wrapper class around boto3 that allows registering of custom credential providers
    and provides methods for setting the default boto3 session or fetching a new session
    with the new context
    """

    def __init__(self) -> None:
        self.session = get_session()
        self._register_credential_provider(provider_name=SharedCredentialProvider.METHOD)

    def _register_credential_provider(self, provider_name: str) -> None:
        credential_chain = self.session.get_component("credential_provider")
        credential_provider = credential_chain.get_provider(provider_name)
        managed_credential_provider = RefreshedCredentialProvider(
            credential_provider=credential_provider
        )
        credential_chain.insert_before(
            name=provider_name, credential_provider=managed_credential_provider
        )

    def get_session(self) -> Session:
        return boto3.Session(botocore_session=self.session)

    def set_default(self) -> None:
        boto3.setup_default_session(botocore_session=self.session)


class RefreshedCredentialProvider(CredentialProvider):  # type: ignore
    """
    Creates a refreshable credential provider class given an existing credential provider in
    the boto3 credential chain.

    """

    METHOD = "managed-refresh-cred"

    def __init__(self, credential_provider: CredentialProvider, refresh_after: int = 60) -> None:
        super().__init__()
        self.refresh_after = timedelta(minutes=refresh_after)
        self.credential_provider = credential_provider

    def load(self) -> Optional[RefreshableCredentials]:
        credentials = self._credentials()
        return credentials and RefreshableCredentials(
            method=self.METHOD,
            refresh_using=self._refresh,
            access_key=credentials.access_key,
            secret_key=credentials.secret_key,
            token=credentials.token,
            expiry_time=self._expiry_time(),
        )

    def _refresh(self) -> Dict:
        credentials = self._credentials()
        expiry_time = self._expiry_time().isoformat()
        logging.info(
            f"credentials {self.credential_provider.CANONICAL_NAME} "
            f"are expiring, refreshing with new expiry time {expiry_time}"
        )
        return {
            "access_key": credentials.access_key,
            "secret_key": credentials.secret_key,
            "token": credentials.token,
            "expiry_time": expiry_time,
        }

    def _credentials(self) -> Credentials:
        return self.credential_provider.load()

    def _expiry_time(self) -> Any:
        return _local_now() + self.refresh_after


def initialize_boto3() -> None:
    boto3_manager = Boto3Manager()
    boto3_manager.set_default()
