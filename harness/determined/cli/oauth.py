import argparse
from typing import Any, List

from determined import cli
from determined.cli import render
from determined.experimental import client


def list_clients(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    oauth_clients = d.list_oauth_clients()
    headers = ["Name", "Client ID", "Domain"]
    keys = ["name", "id", "domain"]
    oauth_clients_dict = [
        {"name": oclient_obj.name, "id": oclient_obj.id, "domain": oclient_obj.domain}
        for oclient_obj in oauth_clients
    ]
    render.tabulate_or_csv(
        headers, [[str(oclient[k]) for k in keys] for oclient in oauth_clients_dict], False
    )


def add_client(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    oauth_client = d.add_oauth_client(domain=args.domain, name=args.name)

    print(f"Client ID:     {oauth_client.id}")
    print(f"Client secret: {oauth_client.secret}")


def remove_client(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    d.remove_oauth_client(client_id=args.client_id)


# fmt: off

args_description = [
    cli.Cmd("oauth", None, "manage OAuth", [
        cli.Cmd("client", None, "manage clients", [
            cli.Cmd("list ls", list_clients, "list OAuth client applications", [], is_default=True),
            cli.Cmd("add", add_client, "add OAuth client application", [
                cli.Arg("name", type=str, help="descriptive name"),
                cli.Arg("domain", type=str, help="redirect domain"),
            ]),
            cli.Cmd("remove", remove_client, "remove OAuth client application", [
                cli.Arg("client_id", help="OAuth client ID to remove"),
            ]),
        ])
    ])
]  # type: List[Any]

# fmt: on
