import argparse
import hashlib
import os
import socket
import ssl
import sys
from typing import List, Sequence, Union, cast
from urllib import parse

import argcomplete
import argcomplete.completers
import requests
import tabulate
import termcolor
from OpenSSL import SSL, crypto

import determined as det
from determined import cli
from determined.cli import (
    agent,
    checkpoint,
    command,
    dev,
    errors,
    experiment,
    job,
    master,
    model,
    notebook,
    oauth,
    project,
    rbac,
    render,
    resource_pool,
    resources,
    shell,
    sso,
    task,
    template,
    tensorboard,
    top_arg_descriptions,
    trial,
    user,
    user_groups,
    version,
    workspace,
)
from determined.common import api, util, yaml
from determined.common.api import bindings, certs


def preview_search(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    experiment_config = util.safe_load_yaml_with_exceptions(args.config_file)
    args.config_file.close()

    if "searcher" not in experiment_config:
        print("Experiment configuration must have 'searcher' section")
        sys.exit(1)
    r = sess.post("searcher/preview", json=experiment_config)
    j = r.json()

    def to_full_name(kind: str) -> str:
        try:
            # The unitless searcher case, for masters newer than 0.17.6.
            length = int(kind)
            return f"train for {length}"
        except ValueError:
            pass
        if kind[-1] == "R":
            return "train {} records".format(kind[:-1])
        if kind[-1] == "B":
            return "train {} batch(es)".format(kind[:-1])
        if kind[-1] == "E":
            return "train {} epoch(s)".format(kind[:-1])
        if kind == "V":
            return "validation"
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

    print(termcolor.colored("Using search configuration:", "green"))
    yml = yaml.YAML()
    yml.indent(mapping=2, sequence=4, offset=2)
    yml.dump(experiment_config["searcher"], sys.stdout)
    print()
    print("This search will create a total of {} trial(s).".format(sum(j["results"].values())))
    print(tabulate.tabulate(values, headers, tablefmt="presto"), flush=False)


args_description = [
    cli.Arg("-u", "--user", help="run as the given user", metavar="username", default=None),
    cli.Arg(
        "-m",
        "--master",
        help="master address",
        metavar="address",
        type=api.canonicalize_master_url,
        default=api.get_default_master_url(),
    ),
    cli.Arg(
        "-v",
        "--version",
        action="version",
        help="print CLI version and exit",
        version="%(prog)s {}".format(det.__version__),
    ),
    cli.Cmd(
        "preview-search",
        preview_search,
        "preview search",
        [
            cli.Arg(
                "config_file", type=argparse.FileType("r"), help="experiment config file (.yaml)"
            )
        ],
    ),
    top_arg_descriptions.deploy_cmd,
]  # type: cli.ArgsDescription

all_args_description: cli.ArgsDescription = (
    args_description
    + experiment.args_description
    + checkpoint.args_description
    + master.args_description
    + model.args_description
    + agent.args_description
    + notebook.args_description
    + job.args_description
    + resources.args_description
    + resource_pool.args_description
    + project.args_description
    + shell.args_description
    + task.args_description
    + template.args_description
    + tensorboard.args_description
    + trial.args_description
    + command.args_description
    + user.args_description
    + user_groups.args_description
    + rbac.args_description
    + version.args_description
    + workspace.args_description
    + sso.args_description
    + oauth.args_description
    + dev.args_description
)


def make_parser() -> argparse.ArgumentParser:
    return argparse.ArgumentParser(
        description="Determined command-line client",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )


def die(message: str, always_print_traceback: bool = False, exit_code: int = 1) -> None:
    if always_print_traceback or util.debug_mode():
        import traceback

        traceback.print_exc(file=sys.stderr)

    print(termcolor.colored(message, "red"), file=sys.stderr, end="\n")
    exit(exit_code)


def main(
    args: List[str] = sys.argv[1:],
) -> None:
    if sys.platform == "win32":
        # Magic incantation to make a Windows 10 cmd.exe process color-related ANSI escape codes.
        os.system("")

    # TODO: we lazily import "det deploy" but in the future we'd want to lazily import everything.
    parser = make_parser()

    full_cmd, aliases = cli.generate_aliases(top_arg_descriptions.deploy_cmd.name)
    is_deploy_cmd = len(args) > 0 and any(args[0] == alias for alias in [*aliases, full_cmd])
    if is_deploy_cmd:
        from determined.deploy import cli as deploy_cli

        cli.add_args(parser, [deploy_cli.args_description])
    else:
        cli.add_args(parser, all_args_description)

    try:
        argcomplete.autocomplete(parser)

        parsed_args = parser.parse_args(args)

        v = vars(parsed_args)
        if not v.get("func"):
            parser.print_usage()
            parser.exit(2, "{}: no subcommand specified\n".format(parser.prog))

        try:
            # For `det deploy`, skip interaction with master.
            if is_deploy_cmd:
                parsed_args.func(parsed_args)
                return

            # Configure the CLI's Cert singleton.
            cli.cert = certs.default_load(parsed_args.master)

            try:
                version.check_version(cli.unauth_session(parsed_args), parsed_args)
            except requests.exceptions.SSLError:
                # An SSLError usually means that we queried a master over HTTPS and got an untrusted
                # cert, so allow the user to store and trust the current cert. (It could also mean
                # that we tried to talk HTTPS on the HTTP port, but distinguishing that based on the
                # exception is annoying, and we'll figure that out in the next step anyway.)
                addr = parse.urlparse(parsed_args.master)
                try:
                    ctx = SSL.Context(SSL.TLSv1_2_METHOD)
                    conn = SSL.Connection(ctx, socket.socket())
                    conn.set_tlsext_host_name(cast(str, addr.hostname).encode())
                    conn.connect(cast(Sequence[Union[str, int]], (addr.hostname, addr.port)))
                    conn.do_handshake()
                    peer_cert_chain = conn.get_peer_cert_chain()
                    if peer_cert_chain is None or len(peer_cert_chain) == 0:
                        # Peer presented no cert.  It seems unlikely that this is possible after
                        # do_handshake() succeeded, but checking for None makes mypy happy.
                        raise crypto.Error()
                    cert_pem_data = [
                        crypto.dump_certificate(crypto.FILETYPE_PEM, cert).decode()
                        for cert in peer_cert_chain
                    ]
                except crypto.Error:
                    die(
                        "Tried to connect over HTTPS but couldn't get a certificate from the "
                        "master; consider using HTTP"
                    )

                # Compute the fingerprint of the certificate; this is the same as the output of
                # `openssl x509 -fingerprint -sha256 -inform pem -noout -in <cert>`.
                cert_hash = hashlib.sha256(ssl.PEM_cert_to_DER_cert(cert_pem_data[0])).hexdigest()
                cert_fingerprint = ":".join(util.chunks(cert_hash, 2))

                if not render.yes_or_no(
                    "The master sent an untrusted certificate chain with this SHA256 fingerprint:\n"
                    "{}\nDo you want to trust this certificate from now on?".format(
                        cert_fingerprint
                    )
                ):
                    die("Unable to verify master certificate")

                joined_certs = "".join(cert_pem_data)
                certs.CertStore(certs.default_store()).set_cert(parsed_args.master, joined_certs)
                # Reconfigure the CLI's Cert singleton, but preserve the certificate name.
                old_cert_name = cli.cert.name
                cli.cert = certs.Cert(cert_pem=joined_certs, name=old_cert_name)

                version.check_version(cli.unauth_session(parsed_args), parsed_args)

            parsed_args.func(parsed_args)
        except KeyboardInterrupt as e:
            raise e
        except (api.errors.BadRequestException, api.errors.BadResponseException) as e:
            die(f"Failed to {parsed_args.func.__name__}: {e}")
        except api.errors.CorruptTokenCacheException:
            die(
                "Failed to login: Attempted to read a corrupted token cache. "
                "The store has been deleted; please try again."
            )
        except det.errors.EnterpriseOnlyError as e:
            die(f"Determined Enterprise Edition is required for this functionality: {e}")
        except errors.FeatureFlagDisabled as e:
            die(f"Master does not support this operation: {e}")
        except errors.CliError as e:
            die(e.message, exit_code=e.exit_code)
        except argparse.ArgumentError as e:
            die(e.message, exit_code=2)
        except bindings.APIHttpError as e:
            die(f"Failed on operation {e.operation_name}: {e.message}")
        except Exception:
            die(f"Failed to {parsed_args.func.__name__}", always_print_traceback=True)
    except KeyboardInterrupt:
        die("Interrupting...", exit_code=3)
