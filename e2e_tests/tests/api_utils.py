from determined.common import api
from determined.common.api import authentication, certs
from tests import config as conf


def determined_test_session() -> api.Session:
    murl = conf.make_master_url()
    certs.cli_cert = certs.default_load(murl)
    authentication.cli_auth = authentication.Authentication(murl)
    return api.Session(murl, "determined", authentication.cli_auth, certs.cli_cert)
