from typing import Optional
from determined.common import api
from determined.common.api import authentication, certs
from tests import config as conf


def determined_test_session(
    credentials: Optional[authentication.Credentials] = None,
) -> api.Session:
    credentials = credentials or authentication.Credentials("determined", "")
    murl = conf.make_master_url()
    certs.cli_cert = certs.default_load(murl)
    authentication.cli_auth = authentication.Authentication(
        murl, requested_user=credentials.username, password=credentials.password
    )
    return api.Session(murl, credentials.username, authentication.cli_auth, certs.cli_cert)
