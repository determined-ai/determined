import uuid
from typing import Optional

from determined.common import api
from determined.common.api import authentication, bindings, certs
from tests import config as conf
from tests.cluster import test_users


def get_random_string() -> str:
    return str(uuid.uuid4())


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


def create_test_user(
    add_password: bool = False,
    session: Optional[api.Session] = None,
    user: Optional[bindings.v1User] = None,
) -> authentication.Credentials:
    session = session or determined_test_session(admin=True)
    user = user or bindings.v1User(username=get_random_string(), admin=False, active=True)
    password = get_random_string() if add_password else ""
    bindings.post_PostUser(session, body=bindings.v1PostUserRequest(user=user, password=password))
    return authentication.Credentials(user.username, password)
