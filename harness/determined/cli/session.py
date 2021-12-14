from argparse import Namespace

from determined.common import util
from determined.common.api import authentication, certs
from determined.common.experimental import session


def setup_session(args: Namespace) -> session.Session:
    master = args.master or util.get_default_master_address()
    cert = certs.default_load(
        master_url=master,
        explicit_path=getattr(args, "cert_path", None),
        explicit_cert_name=getattr(args, "cert_name", None),
        explicit_noverify=getattr(args, "noverify", True),
    )

    return session.Session(master, args.user, authentication.cli_auth, cert)
