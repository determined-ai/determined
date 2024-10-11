import argparse

from determined import cli
from determined.cli import render
from determined.common import api, util
from determined.common.api import bindings


def describe_config_policies(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    workload_type = "EXPERIMENT"
    if args.workload_type.upper() == "NTSC":
        workload_type = "NTSC"

    wksp = api.workspace_by_name(sess, args.workspace)
    resp = bindings.get_GetWorkspaceConfigPolicies(
        sess, workloadType=workload_type, workspaceId=wksp.id
    )
    if args.json:
        render.print_json(resp.configPolicies)
    else:
        print(util.yaml_safe_dump(resp.configPolicies, default_flow_style=False))


def set_config_policies(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        with open(args.config_file, "r") as f:
            data = f.read()
    except Exception as e:
        raise cli.errors.CliError(f"Error opening file: {e}")
    workload_type = "EXPERIMENT"
    if args.workload_type.upper() == "NTSC":
        workload_type = "NTSC"

    wksp = api.workspace_by_name(sess, args.workspace)
    body = bindings.v1PutWorkspaceConfigPoliciesRequest(
        workloadType=workload_type, configPolicies=data, workspaceId=wksp.id
    )
    resp = bindings.put_PutWorkspaceConfigPolicies(
        sess, workloadType=workload_type, workspaceId=wksp.id, body=body
    )
    print(util.yaml_safe_dump(resp.configPolicies, default_flow_style=False))


def delete_config_policies(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    workload_type = "EXPERIMENT"
    if args.workload_type.upper() == "NTSC":
        workload_type = "NTSC"

    wksp = api.workspace_by_name(sess, args.workspace)
    bindings.delete_DeleteWorkspaceConfigPolicies(
        sess, workloadType=workload_type, workspaceId=wksp.id
    )
    print(f"Successfully deleted {workload_type} config policies for workspace {args.workspace}.")


args_description: cli.ArgsDescription = [
    cli.Cmd(
        "config-policies",
        None,
        "manage config policies",
        [
            cli.Cmd(
                "describe",
                describe_config_policies,
                "describe config policies",
                [
                    cli.Arg(
                        "workload_type",
                        type=str,
                        choices=["experiment", "ntsc"],
                        help="the type (Experiment or NTSC ) of config policies",
                    ),
                    cli.Arg(
                        "--workspace",
                        type=str,
                        required=True,  # change to false when adding --global
                        help="config policies for the given workspace",
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
                        choices=["experiment", "ntsc"],
                        help="the type (Experiment or NTSC ) of config policies",
                    ),
                    cli.Arg(
                        "--workspace",
                        type=str,
                        required=True,  # change to false when adding --global
                        help="config policies for the given workspace",
                    ),
                    cli.Arg(
                        "--config-file",
                        type=str,
                        required=True,
                        help="path to the yaml file containing defined config policies",
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
                        choices=["experiment", "ntsc"],
                        help="the type (Experiment or NTSC ) of config policies",
                    ),
                    cli.Arg(
                        "--workspace",
                        type=str,
                        required=True,  # change to false when adding --global
                        help="config policies for the given workspace",
                    ),
                ],
            ),
        ],
    )
]
