from argparse import Namespace

from determined.common import util
from determined.common.api import authentication, certs
from determined.common.experimental import session


def setup_session(args: Namespace) -> session.Session:
    master_url = args.master or util.get_default_master_address()
    cert = certs.default_load(master_url)

    return session.Session(master_url, args.user, authentication.cli_auth, cert)
