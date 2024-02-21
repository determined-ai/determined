"""
tunnel.py will tunnel a TCP connection to the service (typically a shell) with ID equal to
SERVICE_UUID over a WebSocket connection to a Determined master at MASTER_ADDR.
"""

import argparse
import time

from determined.common import api
from determined.common.api import authentication, certs

from .proxy import ListenerConfig, http_connect_tunnel, http_tunnel_listener

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Tunnel through a Determined master")
    parser.add_argument("master_addr")
    parser.add_argument("service_uuid")
    parser.add_argument("--cert-file")
    parser.add_argument("--cert-name")
    parser.add_argument("--listener", type=int)
    parser.add_argument("-u", "--user")
    parser.add_argument("--auth", action="store_true")
    args = parser.parse_args()

    if args.cert_file == "noverify":
        # The special string "noverify" means to not even check the TLS cert.
        cert_file = None
        noverify = True
    else:
        cert_file = args.cert_file
        noverify = False

    cert = certs.default_load(args.master_addr, cert_file, args.cert_name, noverify)

    if args.auth:
        utp = authentication.login_with_cache(args.master_addr, args.user, cert=cert)
        sess: api.BaseSession = api.Session(args.master_addr, utp, cert)
    else:
        sess = api.UnauthSession(args.master_addr, cert)

    if args.listener:
        with http_tunnel_listener(
            sess, [ListenerConfig(service_id=args.service_uuid, local_port=args.listener)]
        ):
            while True:
                time.sleep(1)
    else:
        http_connect_tunnel(sess, args.service_uuid)
