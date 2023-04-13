from argparse import Namespace
from typing import Any, List

from determined.cli import login_sdk_client, render
from determined.common.declarative_argparse import Arg, Cmd
from determined.experimental import client


@login_sdk_client
def list_clients(parsed_args: Namespace) -> None:
    oauth_clients = client.list_oauth_clients()
    headers = ["Name", "Client ID", "Domain"]
    keys = ["name", "id", "domain"]
    oauth_clients_dict = [
        {"name": oclient_obj.name, "id": oclient_obj.id, "domain": oclient_obj.domain}
        for oclient_obj in oauth_clients
    ]
    render.tabulate_or_csv(
        headers, [[str(oclient[k]) for k in keys] for oclient in oauth_clients_dict], False
    )


@login_sdk_client
def add_client(parsed_args: Namespace) -> None:
    oauth_client = client.add_oauth_client(domain=parsed_args.domain, name=parsed_args.name)

    print("Client ID:     {}".format(oauth_client.id))
    print("Client secret: {}".format(oauth_client.secret))


@login_sdk_client
def remove_client(parsed_args: Namespace) -> None:
    client.remove_oauth_client(client_id=parsed_args.client_id)


# fmt: off

args_description = [
    Cmd("oauth", None, "manage OAuth", [
        Cmd("client", None, "manage clients", [
            Cmd("list ls", list_clients, "list OAuth client applications", [], is_default=True),
            Cmd("add", add_client, "add OAuth client application", [
                Arg("name", type=str, help="descriptive name"),
                Arg("domain", type=str, help="redirect domain"),
            ]),
            Cmd("remove", remove_client, "remove OAuth client application", [
                Arg("client_id", help="OAuth client ID to remove"),
            ]),
        ])
    ])
]  # type: List[Any]

# fmt: on
