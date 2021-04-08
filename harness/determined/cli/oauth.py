from argparse import Namespace
from typing import Any, List

from determined.cli import render
from determined.cli.user import authentication_required
from determined.common import api
from determined.common.declarative_argparse import Arg, Cmd


@authentication_required
def list_clients(parsed_args: Namespace) -> None:
    clients = api.get(parsed_args.master, "oauth2/clients").json()
    headers = ["Name", "Client ID", "Domain"]
    keys = ["name", "id", "domain"]
    render.tabulate_or_csv(headers, [[str(client[k]) for k in keys] for client in clients], False)


@authentication_required
def add_client(parsed_args: Namespace) -> None:
    client = api.post(
        parsed_args.master,
        "oauth2/clients",
        body={"domain": parsed_args.domain, "name": parsed_args.name},
    ).json()
    print("Client ID:     {}".format(client["id"]))
    print("Client secret: {}".format(client["secret"]))


@authentication_required
def remove_client(parsed_args: Namespace) -> None:
    api.delete(parsed_args.master, "oauth2/clients/{}".format(parsed_args.client_id))


# fmt: off

args_description = [
    Cmd("oauth", None, "manage OAuth", [
        Cmd("client", None, "manage clients", [
            Cmd("list", list_clients, "list OAuth client applications", [], is_default=True),
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
