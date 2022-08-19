from argparse import Namespace

from determined.common import api, util
from determined.common.api import authentication, certs


def setup_session(args: Namespace) -> api.Session:
    master_url = args.master or util.get_default_master_address()
    cert = certs.default_load(master_url)

    return api.Session(master_url, args.user, authentication.cli_auth, cert)
