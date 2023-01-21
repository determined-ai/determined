from typing import Optional

from determined.common import api
from determined.common.api import authentication, certs
from tests import config as conf
from tests.cluster import test_users


def determined_test_session(
    credentials: Optional[authentication.Credentials] = None,
    admin: Optional[bool] = None,
) -> api.Session:
    assert admin is None or credentials is None, "admin and credentials are mutually exclusive"

    if credentials is None:
        if admin:
            credentials = test_users.ADMIN_CREDENTIALS
        else:
            credentials = authentication.Credentials("determined", "")

    murl = conf.make_master_url()
    certs.cli_cert = certs.default_load(murl)
    authentication.cli_auth = authentication.Authentication(
        murl, requested_user=credentials.username, password=credentials.password
    )
    return api.Session(murl, credentials.username, authentication.cli_auth, certs.cli_cert)
