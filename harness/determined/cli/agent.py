import argparse
import collections
import operator
import os
import sys
import typing
from typing import Any, Callable, Dict, List

import determined.cli.render
from determined import cli
from determined.cli import errors, render
from determined.cli import task as cli_task
from determined.common import check
from determined.common.api import bindings

NO_PERMISSIONS = "NO PERMISSIONS"


def local_id(address: str) -> str:
    return os.path.basename(address)


def list_agents(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    resp = bindings.get_GetAgents(sess)

    agents = [
        collections.OrderedDict(
            [
                ("id", local_id(a.id)),
                ("version", a.version),
                ("registered_time", render.format_time(a.registeredTime)),
                ("num_slots", len(a.slots) if a.slots is not None else ""),
                ("num_containers", len(a.containers) if a.containers is not None else ""),
                (
                    "resource_pools",
                    ", ".join(a.resourcePools) if a.resourcePools is not None else "",
                ),
                ("enabled", a.enabled),
                ("draining", a.draining),
                ("addresses", ", ".join(a.addresses) if a.addresses is not None else ""),
            ]
        )
        for a in sorted(resp.agents or [], key=operator.attrgetter("id"))
    ]

    if args.json:
        determined.cli.render.print_json(agents)
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
        "Addresses",
    ]
    values = [a.values() for a in agents]
    render.tabulate_or_csv(headers, values, args.csv)


def list_slots(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    task_res = bindings.get_GetTasks(sess)
    resp = bindings.get_GetAgents(sess)

    allocations = task_res.allocationIdToSummary

    c_names = (
        {
            r.containerId: {"name": a.name, "allocation_id": a.allocationId}
            for a in allocations.values()
            for r in (a.resources or {})
            if r.containerId
        }
        if allocations
        else {}
    )

    def device_type_string(deviceType: typing.Optional[bindings.devicev1Type]) -> str:
        if deviceType == bindings.devicev1Type.CUDA:
            return "cuda"
        if deviceType == bindings.devicev1Type.ROCM:
            return "rocm"
        if deviceType == bindings.devicev1Type.CPU:
            return "cpu"
        return "unknown"

    def get_task_name(containers: Dict[str, Any], slot: bindings.v1Slot) -> str:
        if not slot.container:
            return "FREE"

        if slot.container.permissionDenied:
            return NO_PERMISSIONS

        container_id = slot.container.id

        if slot.container and container_id in containers:
            return str(containers[container_id]["name"])

        if slot.container and (
            "determined-master-deployment" in container_id
            or "determined-db-deployment" in container_id
        ):
            return f"Determined System Task: {container_id}"

        if slot.container and ("dispatcherrm-inuse-slot-placeholder" in container_id):
            return ""  # slot:task relationship not tracked on HPC clusters, so just show ""

        return f"Non-Determined Task: {container_id}"

    slots = [
        collections.OrderedDict(
            [
                ("agent_id", local_id(agent.id)),
                (
                    "resource_pools",
                    ", ".join(agent.resourcePools) if agent.resourcePools is not None else "",
                ),
                ("slot_id", local_id(slot.id or "")),
                ("enabled", slot.enabled),
                ("draining", slot.draining),
                (
                    "allocation_id",
                    c_names[slot.container.id]["allocation_id"]
                    if slot.container and slot.container.id in c_names
                    else ("OCCUPIED" if slot.container else "FREE"),
                ),
                ("task_name", get_task_name(c_names, slot)),
                ("type", device_type_string((slot.device or bindings.v1Device()).type)),
                ("device", (slot.device or bindings.v1Device()).brand),
            ]
        )
        for agent in sorted(resp.agents or [], key=operator.attrgetter("id"))
        for _key, slot in (agent.slots or {}).items()
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
        determined.cli.render.print_json(slots)
        return

    values = [s.values() for s in slots]

    render.tabulate_or_csv(headers, values, args.csv)


def patch_agent(enabled: bool) -> Callable[[argparse.Namespace], None]:
    def patch(args: argparse.Namespace) -> None:
        sess = cli.setup_session(args)
        check.check_false(args.all and args.agent_id)
        action = "enable" if enabled else "disable"

        if not (args.all or args.agent_id):
            raise errors.CliError(
                "Please pass agent id or --all option. "
                f"See `det agent {action} --help` for details."
            )

        if args.agent_id:
            agent_ids = [args.agent_id]
        else:
            resp = bindings.get_GetAgents(sess)
            agent_ids = sorted(local_id(a.id) for a in resp.agents or [])

        drain_mode = None if enabled else args.drain

        for agent_id in agent_ids:
            path = f"api/v1/agents/{agent_id}/{action}"

            payload = None
            if not enabled and drain_mode:
                payload = {
                    "drain": drain_mode,
                }

            sess.post(path, json=payload)
            status = "Disabled" if not enabled else "Enabled"
            print(f"{status} agent {agent_id}.", file=sys.stderr)

        # When draining, check if there're any tasks currently running on
        # these slots, and list them.
        if drain_mode:
            rsp = bindings.get_GetTasks(sess)
            tasks_data = {
                k: t
                for (k, t) in (
                    rsp.allocationIdToSummary.items()
                    if rsp.allocationIdToSummary is not None
                    else {}
                )
                if any(a in agent_ids for r in (t.resources or []) for a in (r.agentDevices or {}))
            }

            if not (args.json or args.csv):
                if tasks_data:
                    print("Tasks still in progress on draining nodes.")
                else:
                    print("No tasks in progress on draining nodes.")

            cli_task.render_tasks(args, tasks_data)

    return patch


def patch_slot(enabled: bool) -> Callable[[argparse.Namespace], None]:
    def patch(args: argparse.Namespace) -> None:
        sess = cli.setup_session(args)
        if enabled:
            bindings.post_EnableSlot(sess, agentId=args.agent_id, slotId=args.slot_id)
        else:
            bindings.post_DisableSlot(
                sess,
                agentId=args.agent_id,
                slotId=args.slot_id,
                body=bindings.v1DisableSlotRequest(),
            )

        status = "Disabled" if not enabled else "Enabled"
        print("{} slot {} of agent {}".format(status, args.slot_id, args.agent_id))

    return patch


def agent_id_completer(_1: str, parsed_args: argparse.Namespace, _2: Any) -> List[str]:
    resp = bindings.get_GetAgents(cli.setup_session(parsed_args))
    return [a.id for a in resp.agents or []]


# fmt: off

args_description = [
    cli.Cmd("a|gent", None, "manage agents", [
        cli.Cmd("list ls", list_agents, "list agents", [
            cli.Group(
                cli.Arg("--csv", action="store_true", help="print as CSV"),
                cli.Arg("--json", action="store_true", help="print as JSON"),
            ),
        ], is_default=True),
        cli.Cmd("enable", patch_agent(True), "enable agent", [
            cli.Group(
                cli.Arg("agent_id", help="agent ID", nargs="?", completer=agent_id_completer),
                cli.Arg("--all", action="store_true", help="enable all agents"),
            )
        ]),
        cli.Cmd("disable", patch_agent(False), "disable agent", [
            cli.Group(
                cli.Arg("agent_id", help="agent ID", nargs="?", completer=agent_id_completer),
                cli.Arg("--all", action="store_true", help="disable all agents"),
            ),
            cli.Arg(
                "--drain", action="store_true",
                help="enter drain mode, allowing the tasks currently running on "
                "the disabled agents to finish. will also print these tasks, if any"
            ),
            cli.Group(
                cli.Arg("--csv", action="store_true", help="print as CSV"),
                cli.Arg("--json", action="store_true", help="print as JSON"),
            ),
        ]),
    ]),
    cli.Cmd("s|lot", None, "manage slots", [
        cli.Cmd("list ls", list_slots, "list slots in cluster", [
            cli.Group(
                cli.Arg("--csv", action="store_true", help="print as CSV"),
                cli.Arg("--json", action="store_true", help="print as JSON"),
            ),
        ], is_default=True),
        cli.Cmd("enable", patch_slot(True), "enable slot on agent", [
            cli.Arg("agent_id", help="agent ID", completer=agent_id_completer),
            cli.Arg("slot_id", type=int, help="slot ID"),
        ]),
        cli.Cmd("disable", patch_slot(False), "disable slot on agent", [
            cli.Arg("agent_id", help="agent ID", completer=agent_id_completer),
            cli.Arg("slot_id", type=int, help="slot ID"),
        ]),
    ]),
]  # type: List[Any]

# fmt: on
