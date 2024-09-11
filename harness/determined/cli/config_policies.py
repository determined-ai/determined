import argparse

from determined import cli
from determined.cli import render
from determined.common import api, util
from determined.common.api import bindings


def describe_config_policies(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    workload_type = ""
    if args.workload_type.upper() == "EXPERIMENT":
        workload_type = "EXPERIMENT"
    elif args.workload_type.upper() == "NTSC":
        workload_type = "NTSC"
    else:
        raise cli.errors.CliError(
            "Failed to list config policies: Invalid workload type provided."
            + " Valid options: 'experiment' and 'ntsc'"
        )

    wksp = api.workspace_by_name(sess, args.workspace_name)
    resp = bindings.get_GetWorkspaceConfigPolicies(
        sess, workloadType=workload_type, workspaceId=wksp.id
    )
    if args.json:
        render.print_json(resp.configPolicies)
    else:
        print(util.yaml_safe_dump(resp.configPolicies, default_flow_style=False))
    return None


def set_config_policies(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        with open(args.path, "r") as f:
            data = f.read()
    except Exception as e:
        raise cli.errors.CliError(f"Error opening file: {e}")
    workload_type = ""
    if args.workload_type.upper() == "EXPERIMENT":
        workload_type = "EXPERIMENT"
    elif args.workload_type.upper() == "NTSC":
        workload_type = "NTSC"
    else:
        raise cli.errors.CliError(
            "Failed to set config policies: Invalid workload type provided."
            + " Valid options: 'experiment' and 'ntsc'"
        )

    wksp = api.workspace_by_name(sess, args.workspace_name)
    body = bindings.v1PutWorkspaceConfigPoliciesRequest(
        workloadType=workload_type, configPolicies=data, workspaceId=wksp.id
    )
    resp = bindings.put_PutWorkspaceConfigPolicies(
        sess, workloadType=workload_type, workspaceId=wksp.id, body=body
    )
    print(util.yaml_safe_dump(resp.configPolicies, default_flow_style=False))
    return None


def delete_config_policies(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    workload_type = ""
    if args.workload_type.upper() == "EXPERIMENT":
        workload_type = "EXPERIMENT"
    elif args.workload_type.upper() == "NTSC":
        workload_type = "NTSC"
    else:
        raise cli.errors.CliError(
            "Failed to delete config policies: Invalid workload type provided."
            + " Valid options: 'experiment' and 'ntsc'"
        )

    wksp = api.workspace_by_name(sess, args.workspace_name)
    bindings.delete_DeleteWorkspaceConfigPolicies(
        sess, workloadType=workload_type, workspaceId=wksp.id
    )
    print(
        f"Successfully deleted {workload_type} config policies for workspace {args.workspace_name}."
    )
    return None


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
                        "--workspace-name",
                        type=str,
                        required=True,  # change to false when adding --global
                        help="config policies for the given workspace",
                    ),
                    cli.Arg(
                        "--workload-type",
                        type=str,
                        required=True,
                        help="the type (Experiment or NTSC ) of config policies",
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
                        "--workspace-name",
                        type=str,
                        required=True,  # change to false when adding --global
                        help="config policies for the given workspace",
                    ),
                    cli.Arg(
                        "--workload-type",
                        type=str,
                        required=True,
                        help="the type (Experiment or NTSC ) of config policies",
                    ),
                    cli.Arg(
                        "--path",
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
                        "--workspace-name",
                        type=str,
                        required=True,  # change to false when adding --global
                        help="config policies for the given workspace",
                    ),
                    cli.Arg(
                        "--workload-type",
                        type=str,
                        required=True,
                        help="the type (Experiment or NTSC ) of config policies",
                    ),
                ],
            ),
        ],
    )
]
