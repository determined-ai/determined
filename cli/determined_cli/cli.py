import hashlib
import os
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
from ruamel import yaml
from termcolor import colored

import determined_cli
import determined_common.api.authentication as auth
from determined_cli import checkpoint, experiment, render
from determined_cli.agent import args_description as agent_args_description
from determined_cli.declarative_argparse import Arg, Cmd, add_args
from determined_cli.experiment import args_description as experiment_args_description
from determined_cli.master import args_description as master_args_description
from determined_cli.model import args_description as model_args_description
from determined_cli.notebook import args_description as notebook_args_description
from determined_cli.remote import args_description as remote_args_description
from determined_cli.shell import args_description as shell_args_description
from determined_cli.template import args_description as template_args_description
from determined_cli.tensorboard import args_description as tensorboard_args_description
from determined_cli.trial import args_description as trial_args_description
from determined_cli.user import args_description as user_args_description
from determined_cli.version import args_description as version_args_description
from determined_cli.version import check_version
from determined_common import api
from determined_common.api.authentication import authentication_required
from determined_common.check import check_not_none
from determined_common.util import chunks, debug_mode, get_default_master_address


@authentication_required
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

    def get_state_rank(state: str) -> int:
        if state == "PENDING":
            return 0
        if state == "RUNNING":
            return 1
        if state == "TERMINATING":
            return 2
        if state == "TERMINATED":
            return 3
        return 4

    tasks = r.json()
    headers = ["ID", "Name", "Slots Needed", "Registered Time", "State", "Agent", "Exit Status"]
    values = [
        [
            task_id,
            task["name"],
            task["slots_needed"],
            render.format_time(task["registered_time"]),
            task["state"],
            agent_info(task),
            task["exit_status"] if task.get("exit_status", None) else "N/A",
        ]
        for task_id, task in sorted(
            tasks.items(),
            key=lambda tup: (
                get_state_rank(tup[1]["state"]),
                render.format_time(tup[1]["registered_time"]),
            ),
        )
    ]

    render.tabulate_or_csv(headers, values, args.csv)


@authentication_required
def preview_search(args: Namespace) -> None:
    experiment_config = yaml.safe_load(args.config_file.read())
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
        elif kind == "C":
            return "checkpoint"
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
        version="%(prog)s {}".format(determined_cli.__version__)),

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
]  # type: List[object]

# fmt: on


all_args_description = (
    args_description
    + master_args_description
    + model_args_description
    + agent_args_description
    + notebook_args_description
    + shell_args_description
    + template_args_description
    + tensorboard_args_description
    + trial_args_description
    + remote_args_description
    + user_args_description
    + version_args_description
)


def make_parser() -> ArgumentParser:
    parser = ArgumentParser(
        description="Determined command-line client", formatter_class=ArgumentDefaultsHelpFormatter
    )
    add_args(parser, all_args_description)
    return parser


def main(args: List[str] = sys.argv[1:]) -> None:
    # TODO(#1690): Refactor admin command(s) to a separate CLI tool.
    if "DET_ADMIN" in os.environ:
        experiment_args_description.subs.append(
            Cmd(
                "delete",
                experiment.delete_experiment,
                "delete experiment",
                [
                    Arg("experiment_id", help="delete experiment"),
                    Arg(
                        "--yes",
                        action="store_true",
                        default=False,
                        help="automatically answer yes to prompts",
                    ),
                ],
            )
        )

    try:
        parser = make_parser()
        argcomplete.autocomplete(parser)

        parsed_args = parser.parse_args(args)

        def die(message: str, always_print_traceback: bool = False) -> None:
            if always_print_traceback or debug_mode():
                import traceback

                traceback.print_exc()

            parser.exit(1, colored(message + "\n", "red"))

        v = vars(parsed_args)
        if not v.get("func"):
            parser.print_usage()
            parser.exit(2, "{}: no subcommand specified\n".format(parser.prog))

        cert_fn = str(auth.get_config_path().joinpath("master.crt"))
        if os.path.exists(cert_fn):
            api.request.set_master_cert_bundle(cert_fn)

        try:
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

                with open(cert_fn, "w") as out:
                    out.write(cert_pem_data)
                api.request.set_master_cert_bundle(cert_fn)

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
        except Exception:
            die("Failed to {}".format(parsed_args.func.__name__), always_print_traceback=True)
    except KeyboardInterrupt:
        parser.exit(3, colored("Interrupting...\n", "red"))
