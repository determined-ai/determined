import hashlib
import socket
import ssl
import sys
from argparse import ArgumentDefaultsHelpFormatter, ArgumentParser, FileType, Namespace
from typing import Any, Dict, List, Union, cast

import argcomplete
import argcomplete.completers
import OpenSSL
import requests
import tabulate
from termcolor import colored

import determined
import determined.cli
from determined.cli import checkpoint, experiment, render
from determined.cli.agent import args_description as agent_args_description
from determined.cli.master import args_description as master_args_description
from determined.cli.model import args_description as model_args_description
from determined.cli.notebook import args_description as notebook_args_description
from determined.cli.oauth import args_description as oauth_args_description
from determined.cli.remote import args_description as remote_args_description
from determined.cli.resources import args_description as resources_args_description
from determined.cli.shell import args_description as shell_args_description
from determined.cli.sso import args_description as auth_args_description
from determined.cli.template import args_description as template_args_description
from determined.cli.tensorboard import args_description as tensorboard_args_description
from determined.cli.trial import args_description as trial_args_description
from determined.cli.user import args_description as user_args_description
from determined.cli.version import args_description as version_args_description
from determined.cli.version import check_version
from determined.common import api, yaml
from determined.common.api import authentication, certs
from determined.common.check import check_not_none
from determined.common.declarative_argparse import Arg, Cmd, add_args
from determined.common.util import (
    chunks,
    debug_mode,
    get_default_master_address,
    safe_load_yaml_with_exceptions,
)
from determined.deploy.cli import DEPLOY_CMD_NAME
from determined.deploy.cli import args_description as deploy_args_description

from .errors import EnterpriseOnlyError


@authentication.required
def list_tasks(args: Namespace) -> None:
    r = api.get(args.master, "tasks")

    def agent_info(t: Dict[str, Any]) -> Union[str, List[str]]:
        containers = t.get("containers", [])
        if not containers:
            return "unassigned"
        if len(containers) == 1:
            agent = containers[0]["agent"]  # type: str
            return agent
        return [c["agent"] for c in containers]

    tasks = r.json()
    headers = [
        "ID",
        "Name",
        "Slots Needed",
        "Registered Time",
        "Agent",
        "Priority",
        "Resource Pool",
    ]
    values = [
        [
            task["id"],
            task["name"],
            task["slots_needed"],
            render.format_time(task["registered_time"]),
            agent_info(task),
            task["priority"] if task["scheduler_type"] == "priority" else "N/A",
            task["resource_pool"],
        ]
        for task_id, task in sorted(
            tasks.items(),
            key=lambda tup: (render.format_time(tup[1]["registered_time"]),),
        )
    ]

    render.tabulate_or_csv(headers, values, args.csv)


@authentication.required
def preview_search(args: Namespace) -> None:
    experiment_config = safe_load_yaml_with_exceptions(args.config_file)
    args.config_file.close()

    if "searcher" not in experiment_config:
        print("Experiment configuration must have 'searcher' section")
        sys.exit(1)
    r = api.post(args.master, "searcher/preview", body=experiment_config)
    j = r.json()

    def to_full_name(kind: str) -> str:
        if kind[-1] == "R":
            return "train {} records".format(kind[:-1])
        if kind[-1] == "B":
            return "train {} batch(es)".format(kind[:-1])
        if kind[-1] == "E":
            return "train {} epoch(s)".format(kind[:-1])
        elif kind == "V":
            return "validation"
        else:
            raise ValueError("unexpected kind: {}".format(kind))

    def render_sequence(sequence: List[str]) -> str:
        if not sequence:
            return "N/A"
        instructions = []
        current = sequence[0]
        count = 0
        for k in sequence:
            if k != current:
                instructions.append("{} x {}".format(count, to_full_name(current)))
                current = k
                count = 1
            else:
                count += 1
        instructions.append("{} x {}".format(count, to_full_name(current)))
        return ", ".join(instructions)

    headers = ["Trials", "Breakdown"]
    values = [
        (count, render_sequence(operations.split())) for operations, count in j["results"].items()
    ]

    print(colored("Using search configuration:", "green"))
    yml = yaml.YAML()
    yml.indent(mapping=2, sequence=4, offset=2)
    yml.dump(experiment_config["searcher"], sys.stdout)
    print()
    print("This search will create a total of {} trial(s).".format(sum(j["results"].values())))
    print(tabulate.tabulate(values, headers, tablefmt="presto"), flush=False)


# fmt: off

args_description = [
    Arg("-u", "--user",
        help="run as the given user", metavar="username",
        default=None),
    Arg("-m", "--master",
        help="master address", metavar="address",
        default=get_default_master_address()),
    Arg("-v", "--version",
        action="version", help="print CLI version and exit",
        version="%(prog)s {}".format(determined.__version__)),

    experiment.args_description,

    checkpoint.args_description,

    Cmd("task", None, "manage tasks (commands, experiments, notebooks, shells, tensorboards)", [
        Cmd("list", list_tasks, "list tasks in cluster", [
            Arg("--csv", action="store_true", help="print as CSV"),
        ], is_default=True),
    ]),

    Cmd("preview-search", preview_search, "preview search", [
        Arg("config_file", type=FileType("r"),
            help="experiment config file (.yaml)")
    ]),

    deploy_args_description,
]  # type: List[object]

# fmt: on


all_args_description = (
    args_description
    + master_args_description
    + model_args_description
    + agent_args_description
    + notebook_args_description
    + resources_args_description
    + shell_args_description
    + template_args_description
    + tensorboard_args_description
    + trial_args_description
    + remote_args_description
    + user_args_description
    + version_args_description
    + auth_args_description
    + oauth_args_description
)


def make_parser() -> ArgumentParser:
    parser = ArgumentParser(
        description="Determined command-line client", formatter_class=ArgumentDefaultsHelpFormatter
    )
    add_args(parser, all_args_description)
    return parser


def main(args: List[str] = sys.argv[1:]) -> None:
    try:
        parser = make_parser()
        argcomplete.autocomplete(parser)

        parsed_args = parser.parse_args(args)

        def die(message: str, always_print_traceback: bool = False) -> None:
            if always_print_traceback or debug_mode():
                import traceback

                traceback.print_exc(file=sys.stderr)

            parser.exit(1, colored(message + "\n", "red"))

        v = vars(parsed_args)
        if not v.get("func"):
            parser.print_usage()
            parser.exit(2, "{}: no subcommand specified\n".format(parser.prog))

        # Configure the CLI's Cert singleton.
        certs.cli_cert = certs.default_load(parsed_args.master)

        try:
            # For `det deploy`, skip interaction with master.
            if v.get("_command") == DEPLOY_CMD_NAME:
                parsed_args.func(parsed_args)
                return

            try:
                check_version(parsed_args)
            except requests.exceptions.SSLError:
                # An SSLError usually means that we queried a master over HTTPS and got an untrusted
                # cert, so allow the user to store and trust the current cert. (It could also mean
                # that we tried to talk HTTPS on the HTTP port, but distinguishing that based on the
                # exception is annoying, and we'll figure that out in the next step anyway.)
                addr = api.parse_master_address(parsed_args.master)
                check_not_none(addr.hostname)
                check_not_none(addr.port)
                try:
                    ctx = OpenSSL.SSL.Context(OpenSSL.SSL.TLSv1_2_METHOD)
                    conn = OpenSSL.SSL.Connection(ctx, socket.socket())
                    conn.set_tlsext_host_name(cast(str, addr.hostname).encode())
                    conn.connect((addr.hostname, addr.port))
                    conn.do_handshake()
                    cert_pem_data = "".join(
                        OpenSSL.crypto.dump_certificate(OpenSSL.crypto.FILETYPE_PEM, cert).decode()
                        for cert in conn.get_peer_cert_chain()
                    )
                except OpenSSL.SSL.Error:
                    die(
                        "Tried to connect over HTTPS but couldn't get a certificate from the "
                        "master; consider using HTTP"
                    )

                cert_hash = hashlib.sha256(ssl.PEM_cert_to_DER_cert(cert_pem_data)).hexdigest()
                cert_fingerprint = ":".join(chunks(cert_hash, 2))

                if not render.yes_or_no(
                    "The master sent an untrusted certificate chain with this SHA256 fingerprint:\n"
                    "{}\nDo you want to trust this certificate from now on?".format(
                        cert_fingerprint
                    )
                ):
                    die("Unable to verify master certificate")

                certs.CertStore(certs.default_store()).set_cert(parsed_args.master, cert_pem_data)
                # Reconfigure the CLI's Cert singleton, but preserve the certificate name.
                old_cert_name = certs.cli_cert.name
                certs.cli_cert = certs.Cert(cert_pem=cert_pem_data, name=old_cert_name)

                check_version(parsed_args)

            parsed_args.func(parsed_args)
        except KeyboardInterrupt as e:
            raise e
        except (api.errors.BadRequestException, api.errors.BadResponseException) as e:
            die("Failed to {}: {}".format(parsed_args.func.__name__, e))
        except api.errors.CorruptTokenCacheException:
            die(
                "Failed to login: Attempted to read a corrupted token cache. "
                "The store has been deleted; please try again."
            )
        except EnterpriseOnlyError as e:
            die(f"Determined Enterprise Edition is required for this functionality: {e}")
        except Exception:
            die("Failed to {}".format(parsed_args.func.__name__), always_print_traceback=True)
    except KeyboardInterrupt:
        # die() may not be defined yet.
        if debug_mode():
            import traceback

            traceback.print_exc(file=sys.stderr)

        print(colored("Interrupting...\n", "red"), file=sys.stderr)
        exit(3)
