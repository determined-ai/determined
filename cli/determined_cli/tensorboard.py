import sys
from argparse import ONE_OR_MORE, Namespace
from collections import namedtuple
from typing import Any, Dict, List

from termcolor import colored

from determined_common import api
from determined_common.check import check_eq

from . import render
from .command import Command, render_event_stream
from .declarative_argparse import Arg, Cmd
from .user import authentication_required

Tensorboard = namedtuple(
    "Tensorboard",
    ["id", "owner", "description", "state", "experiment_ids", "trial_ids", "exit_status"],
)


def to_tensorboard(command: Command) -> Tensorboard:
    return Tensorboard(
        command.id,
        command.owner["username"],
        command.config["description"],
        command.state,
        command.misc.get("experiment_ids"),
        command.misc.get("trial_ids"),
        command.exit_status,
    )


@authentication_required
def start_tensorboard(args: Namespace) -> None:
    if args.trial_ids is None and args.experiment_ids is None:
        print("Either experiment_ids or trial_ids must be specified.")
        sys.exit(1)

    req_body = {"trial_ids": args.trial_ids, "experiment_ids": args.experiment_ids}
    resp = api.post(args.master, "tensorboard", body=req_body)

    res_body = resp.json()

    if args.detach:
        print(res_body["id"])
        return

    url = "tensorboard/{}/events".format(res_body["id"])
    with api.ws(args.master, url) as ws:
        for msg in ws:
            if msg["log_event"] is not None:
                # TensorBoard will print a url by default. The URL is incorrect since
                # TensorBoard is not aware of the master proxy address it is assigned.
                if "http" in msg["log_event"]:
                    continue

            if msg["service_ready_event"]:
                if args.no_browser:
                    url = api.make_url(args.master, res_body["service_address"])
                else:
                    url = api.open(args.master, res_body["service_address"])

                print(colored("TensorBoard is running at: {}".format(url), "green"))
                render_event_stream(msg)
                break
            render_event_stream(msg)


@authentication_required
def open_tensorboard(args: Namespace) -> None:
    resp = api.get(args.master, "tensorboard/{}".format(args.tensorboard_id)).json()
    tensorboard = render.unmarshal(Command, resp)
    check_eq(tensorboard.state, "RUNNING", "TensorBoard must be in a running state")
    api.open(args.master, resp["service_address"])


@authentication_required
def tail_tensorboard_logs(args: Namespace) -> None:
    url = "tensorboard/{}/events?follow={}&tail={}".format(
        args.tensorboard_id, args.follow, args.tail
    )
    with api.ws(args.master, url) as ws:
        for msg in ws:
            render_event_stream(msg)


@authentication_required
def list_tensorboards(args: Namespace) -> None:
    if args.all:
        params = {}  # type: Dict[str, Any]
    else:
        params = {"user": api.Authentication.instance().get_session_user()}

    commands = [
        render.unmarshal(Command, command)
        for command in api.get(args.master, "tensorboard", params=params).json().values()
    ]

    if args.quiet:
        for command in commands:
            print(command.id)
        return

    render.render_objects(Tensorboard, [to_tensorboard(command) for command in commands])


@authentication_required
def kill_tensorboard(args: Namespace) -> None:
    for i, tid in enumerate(args.tensorboard_id):
        try:
            api.delete(args.master, "tensorboard/{}".format(tid))
            print(colored("Killed tensorboard {}".format(tid), "green"))
        except api.errors.APIException as e:
            if not args.force:
                for ignored in args.tensorboard_id[i + 1 :]:
                    print("Cowardly not killing {}".format(ignored))
                raise e
            print(colored("Skipping: {} ({})".format(e, type(e).__name__), "red"))


# fmt: off

args_description = [
    Cmd("tensorboard", None, "manage TensorBoard instances", [
        Cmd("list ls", list_tensorboards, "list TensorBoard instances", [
            Arg("-q", "--quiet", action="store_true",
                help="only display the IDs"),
            Arg("--all", "-a", action="store_true",
                help="show all TensorBoards (including other users')")
        ], is_default=True),
        Cmd("start", start_tensorboard, "start new TensorBoard instance", [
            Arg("experiment_ids", type=int, nargs="*",
                help="experiment IDs to load into TensorBoard. At most 100 trials from "
                     "the specified experiment will be loaded into TensorBoard. If the "
                     "experiment has more trials, the 100 best-performing trials will "
                     "be used."),
            Arg("-t", "--trial-ids", nargs=ONE_OR_MORE, type=int,
                help="trial IDs to load into TensorBoard; at most 100 trials are "
                     "allowed per TensorBoard instance"),
            Arg("--no-browser", action="store_true",
                help="don't open TensorBoard in a browser after startup"),
            Arg("-d", "--detach", action="store_true",
                help="run in the background and print the ID")
        ]),
        Cmd("open", open_tensorboard,
            "open existing TensorBoard instance", [
                Arg("tensorboard_id", help="TensorBoard ID")
            ]),
        Cmd("logs", tail_tensorboard_logs, "fetch TensorBoard instance logs", [
            Arg("tensorboard_id", help="TensorBoard ID"),
            Arg("-f", "--follow", action="store_true",
                help="follow the logs of a TensorBoard instance, "
                     "similar to tail -f"),
            Arg("--tail", type=int, default=200,
                help="number of lines to show, counting from the end "
                     "of the log")
        ]),
        Cmd("kill", kill_tensorboard, "kill TensorBoard instance", [
            Arg("tensorboard_id", help="TensorBoard ID", nargs=ONE_OR_MORE),
            Arg("-f", "--force", action="store_true", help="ignore errors"),
        ]),
    ])
]  # type: List[Any]

# fmt: on
