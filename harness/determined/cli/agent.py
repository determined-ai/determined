import argparse
import json
import os
import sys
from collections import OrderedDict
from typing import Any, Callable, List

from determined.cli import render
from determined.cli import task as cli_task
from determined.common import api
from determined.common.api import authentication
from determined.common.check import check_false
from determined.common.declarative_argparse import Arg, Cmd, Group


def local_id(address: str) -> str:
    return os.path.basename(address)


@authentication.required
def list_agents(args: argparse.Namespace) -> None:
    r = api.get(args.master, "agents")

    agents = r.json()
    agents = [
        OrderedDict(
            [
                ("id", local_id(agent_id)),
                ("version", agent["version"]),
                ("registered_time", render.format_time(agent["registered_time"])),
                ("num_slots", len(agent["slots"])),
                ("num_containers", agent["num_containers"]),
                ("resource_pool", agent["resource_pool"]),
                ("enabled", agent.get("enabled")),
                ("draining", agent.get("draining")),
                ("label", agent["label"]),
                ("addresses", ", ".join(agent["addresses"])),
            ]
        )
        for agent_id, agent in sorted(agents.items())
    ]

    if args.json:
        print(json.dumps(agents, indent=4))
        return

    headers = [
        "Agent ID",
        "Version",
        "Registered Time",
        "Slots",
        "Containers",
        "Resource Pool",
        "Enabled",
        "Draining",
        "Label",
        "Addresses",
    ]
    values = [a.values() for a in agents]

    render.tabulate_or_csv(headers, values, args.csv)


@authentication.required
def list_slots(args: argparse.Namespace) -> None:
    task_res = api.get(args.master, "tasks")
    agent_res = api.get(args.master, "agents")

    agents = agent_res.json()
    allocations = task_res.json()

    c_names = {
        r["container_id"]: {"name": a["name"], "allocation_id": a["allocation_id"]}
        for a in allocations.values()
        for r in a["resources"]
        if r["container_id"]
    }

    slots = [
        OrderedDict(
            [
                ("agent_id", local_id(agent_id)),
                ("resource_pool", agent["resource_pool"]),
                ("slot_id", local_id(slot_id)),
                ("enabled", slot["enabled"]),
                ("draining", slot.get("draining", False)),
                (
                    "allocation_id",
                    c_names[slot["container"]["id"]]["allocation_id"]
                    if slot["container"]
                    else "FREE",
                ),
                (
                    "task_name",
                    c_names[slot["container"]["id"]]["name"] if slot["container"] else "None",
                ),
                ("type", slot["device"]["type"]),
                ("device", slot["device"]["brand"]),
            ]
        )
        for agent_id, agent in sorted(agents.items())
        for slot_id, slot in sorted(agent["slots"].items())
    ]

    headers = [
        "Agent ID",
        "Resource Pool",
        "Slot ID",
        "Enabled",
        "Draining",
        "Allocation ID",
        "Task Name",
        "Type",
        "Device",
    ]

    if args.json:
        print(json.dumps(slots, indent=4))
        return

    values = [s.values() for s in slots]

    render.tabulate_or_csv(headers, values, args.csv)


def patch_agent(enabled: bool) -> Callable[[argparse.Namespace], None]:
    @authentication.required
    def patch(args: argparse.Namespace) -> None:
        check_false(args.all and args.agent_id)

        if not (args.all or args.agent_id):
            print("Error: must specify exactly one of `--all` or agent_id", file=sys.stderr)
            sys.exit(1)

        if args.agent_id:
            agent_ids = [args.agent_id]
        else:
            r = api.get(args.master, "agents")
            agent_ids = sorted(local_id(a) for a in r.json().keys())

        drain_mode = None if enabled else args.drain

        for agent_id in agent_ids:
            action = "enable" if enabled else "disable"
            path = f"api/v1/agents/{agent_id}/{action}"

            payload = None
            if not enabled and drain_mode:
                payload = {
                    "drain": drain_mode,
                }

            api.post(args.master, path, payload)
            status = "Disabled" if not enabled else "Enabled"
            print(f"{status} agent {agent_id}.", file=sys.stderr)

        # When draining, check if there're any tasks currently running on
        # these slots, and list them.
        if drain_mode:
            rsp = api.get(args.master, "tasks")
            tasks_data = {
                k: t
                for (k, t) in rsp.json().items()
                if any(a in agent_ids for r in t.get("resources", []) for a in r["agent_devices"])
            }

            if not (args.json or args.csv):
                if tasks_data:
                    print("Tasks still in progress on draining nodes.")
                else:
                    print("No tasks in progress on draining nodes.")

            cli_task.render_tasks(args, tasks_data)

    return patch


def patch_slot(enabled: bool) -> Callable[[argparse.Namespace], None]:
    @authentication.required
    def patch(args: argparse.Namespace) -> None:
        path = "agents/{}/slots/{}".format(args.agent_id, args.slot_id)
        headers = {"Content-Type": "application/merge-patch+json"}
        payload = {"enabled": enabled}

        api.patch(args.master, path, json=payload, headers=headers)
        status = "Disabled" if not enabled else "Enabled"
        print("{} slot {} of agent {}".format(status, args.slot_id, args.agent_id))

    return patch


def agent_id_completer(_1: str, parsed_args: argparse.Namespace, _2: Any) -> List[str]:
    r = api.get(parsed_args.master, "agents")
    return list(r.json().keys())


# fmt: off

args_description = [
    Cmd("a|gent", None, "manage agents", [
        Cmd("list", list_agents, "list agents", [
            Group(
                Arg("--csv", action="store_true", help="print as CSV"),
                Arg("--json", action="store_true", help="print as JSON"),
            ),
        ], is_default=True),
        Cmd("enable", patch_agent(True), "enable agent", [
            Group(
                Arg("agent_id", help="agent ID", nargs="?", completer=agent_id_completer),
                Arg("--all", action="store_true", help="enable all agents"),
            )
        ]),
        Cmd("disable", patch_agent(False), "disable agent", [
            Group(
                Arg("agent_id", help="agent ID", nargs="?", completer=agent_id_completer),
                Arg("--all", action="store_true", help="disable all agents"),
            ),
            Arg("--drain", action="store_true",
                help="enter drain mode, allowing the tasks currently running on "
                     "the disabled agents to finish. will also print these tasks, if any"),
            Group(
                Arg("--csv", action="store_true", help="print as CSV"),
                Arg("--json", action="store_true", help="print as JSON"),
            ),
        ]),
    ]),
    Cmd("s|lot", None, "manage slots", [
        Cmd("list", list_slots, "list slots in cluster", [
            Group(
                Arg("--csv", action="store_true", help="print as CSV"),
                Arg("--json", action="store_true", help="print as JSON"),
            ),
        ], is_default=True),
        Cmd("enable", patch_slot(True), "enable slot on agent", [
            Arg("agent_id", help="agent ID", completer=agent_id_completer),
            Arg("slot_id", type=int, help="slot ID"),
        ]),
        Cmd("disable", patch_slot(False), "disable slot on agent", [
            Arg("agent_id", help="agent ID", completer=agent_id_completer),
            Arg("slot_id", type=int, help="slot ID"),
        ]),
    ]),
]  # type: List[Any]

# fmt: on
