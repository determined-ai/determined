import argparse

from determined import cli
from determined.cli import render
from determined.common import api, util
from determined.common.api import bindings


def describe_config_policies(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)

    workload_type = "EXPERIMENT"
    if args.workload_type.upper() == "TASKS":
        workload_type = "NTSC"

    if args.workspace_name:
        wksp = api.workspace_by_name(sess, args.workspace_name)
        wksp_resp = bindings.get_GetWorkspaceConfigPolicies(
            sess, workloadType=workload_type, workspaceId=wksp.id
        )
        if args.json:
            render.print_json(wksp_resp.configPolicies)
        else:
            print(util.yaml_safe_dump(wksp_resp.configPolicies, default_flow_style=False))
        return

    global_resp = bindings.get_GetGlobalConfigPolicies(sess, workloadType=workload_type)
    if args.json:
        render.print_json(global_resp.configPolicies)
    else:
        print(util.yaml_safe_dump(global_resp.configPolicies, default_flow_style=False))


def set_config_policies(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)

    try:
        with open(args.config_file, "r") as f:
            data = f.read()
    except Exception as e:
        raise cli.errors.CliError(f"Error opening file: {e}")
    workload_type = "EXPERIMENT"
    if args.workload_type.upper() == "TASKS":
        workload_type = "NTSC"

    if args.workspace_name:
        wksp = api.workspace_by_name(sess, args.workspace_name)
        wksp_body = bindings.v1PutWorkspaceConfigPoliciesRequest(
            workloadType=workload_type, configPolicies=data, workspaceId=wksp.id
        )
        wksp_resp = bindings.put_PutWorkspaceConfigPolicies(
            sess, workloadType=workload_type, workspaceId=wksp.id, body=wksp_body
        )
        print(f"Set {args.workload_type} config policies for workspace {args.workspace_name}:")
        print(util.yaml_safe_dump(wksp_resp.configPolicies, default_flow_style=False))
        return

    global_body = bindings.v1PutGlobalConfigPoliciesRequest(
        workloadType=workload_type, configPolicies=data
    )
    global_resp = bindings.put_PutGlobalConfigPolicies(
        sess, workloadType=workload_type, body=global_body
    )
    print(f"Set global {args.workload_type} config policies:")
    print(util.yaml_safe_dump(global_resp.configPolicies, default_flow_style=False))


def delete_config_policies(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)

    workload_type = "EXPERIMENT"
    if args.workload_type.upper() == "TASKS":
        workload_type = "NTSC"

    if args.workspace_name:
        wksp = api.workspace_by_name(sess, args.workspace_name)

        bindings.delete_DeleteWorkspaceConfigPolicies(
            sess, workloadType=workload_type, workspaceId=wksp.id
        )
        print(
            f"Successfully deleted {workload_type} config policies for workspace "
            f"{args.workspace_name}"
        )
        return

    bindings.delete_DeleteGlobalConfigPolicies(sess, workloadType=workload_type)
    print(f"Successfully deleted global {workload_type} config policies")


args_description: cli.ArgsDescription = [
    cli.Cmd(
        "config-policies cp",
        None,
        "manage config policies",
        [
            cli.Cmd(
                "describe",
                describe_config_policies,
                "display config policies",
                [
                    cli.Arg(
                        "workload_type",
                        type=str,
                        choices=["experiment", "tasks"],
                        help="the type (Experiment or Tasks) of config policies",
                    ),
                    cli.Arg(
                        "-w",
                        "--workspace-name",
                        type=str,
                        required=False,
                        help="get config policies from this workspace. When not specified, get "
                        "global config policies",
                    ),
                    cli.Group(cli.output_format_args["json"], cli.output_format_args["yaml"]),
                ],
                is_default=True,
            ),
            cli.Cmd(
                "set",
                set_config_policies,
                "set config policies",
                [
                    cli.Arg(
                        "workload_type",
                        type=str,
                        choices=["experiment", "tasks"],
                        help="the type (Experiment or Tasks) of config policies",
                    ),
                    cli.Arg(
                        "config_file",
                        type=str,
                        help="path to the YAML or JSON file containing defined config policies "
                        "(can be absolute or relative to the current directory)",
                    ),
                    cli.Arg(
                        "-w",
                        "--workspace-name",
                        type=str,
                        required=False,
                        help="apply config policies to this workspace. When not specified, apply "
                        "global config policies",
                    ),
                ],
            ),
            cli.Cmd(
                "delete",
                delete_config_policies,
                "delete config policies",
                [
                    cli.Arg(
                        "workload_type",
                        type=str,
                        choices=["experiment", "tasks"],
                        help="the type (Experiment or Tasks) of config policies",
                    ),
                    cli.Arg(
                        "-w",
                        "--workspace-name",
                        type=str,
                        required=False,
                        help="delete config policies from this workspace. When not specified, "
                        "delete global config policies",
                    ),
                ],
            ),
        ],
    )
]
