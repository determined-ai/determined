"""
tunnel.py will tunnel a TCP connection to the service (typically a shell) with ID equal to
SERVICE_UUID over a WebSocket connection to a Determined master at MASTER_ADDR.
"""

import argparse
import time

from determined.common.api import authentication

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

    authorization_token = None
    if args.auth:
        auth = authentication.Authentication(args.master_addr, args.user)
        authorization_token = auth.get_session_token(must=True)

    # The special string "noverify" is passed to our certs.Cert object as a boolean False.
    cert_file = False if args.cert_file == "noverify" else args.cert_file

    if args.listener:
        with http_tunnel_listener(
            args.master_addr,
            [ListenerConfig(service_id=args.service_uuid, local_port=args.listener)],
            cert_file,
            args.cert_name,
            authorization_token,
        ):
            while True:
                time.sleep(1)
    else:
        http_connect_tunnel(
            args.master_addr, args.service_uuid, cert_file, args.cert_name, authorization_token
        )
