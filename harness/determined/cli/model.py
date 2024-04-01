import argparse
import json
from typing import Any, List

from determined import cli
from determined.cli import render, workspace
from determined.common import api
from determined.experimental import client


def render_model(model: client.Model) -> None:
    table = [
        ["ID", model.model_id],
        ["Name", model.name],
        ["Description", model.description],
        ["Workspace ID", model.workspace_id],
        ["Creation Time", model.creation_time],
        ["Last Updated Time", model.last_updated_time],
        ["Metadata", json.dumps(model.metadata or {}, indent=4)],
    ]

    headers, values = zip(*table)  # type: ignore

    render.tabulate_or_csv(headers, [values], False)


def _render_model_versions(model_versions: List[client.ModelVersion]) -> None:
    headers = [
        "Version #",
        "Trial ID",
        "Batch #",
        "Checkpoint UUID",
        "Validation Metrics",
        "Metadata",
    ]

    values = []
    for model_version in model_versions:
        assert model_version.checkpoint
        checkpoint = model_version.checkpoint
        values.append(
            [
                model_version.model_version,
                checkpoint.training.trial_id if checkpoint.training else None,
                checkpoint.metadata["steps_completed"] if checkpoint.metadata else None,
                checkpoint.uuid,
                (
                    json.dumps(checkpoint.training.validation_metrics, indent=2)
                    if checkpoint.training
                    else ""
                ),
                json.dumps(checkpoint.metadata, indent=2),
            ]
        )

    render.tabulate_or_csv(headers, values, False)


def list_models(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    workspace_names = None
    if args.workspace_names is not None:
        workspace_names = args.workspace_names.split(",")
    models = d.list_models(
        sort_by=client.ModelSortBy[args.sort_by.upper()],
        order_by=client.OrderBy[args.order_by.upper()],
        workspace_names=workspace_names,
    )
    if args.json:
        render.print_json([render.model_to_json(m) for m in models])
    else:
        headers = ["ID", "Name", "Workspace ID", "Creation Time", "Last Updated Time", "Metadata"]

        values = [
            [
                m.model_id,
                m.name,
                m.workspace_id,
                m.creation_time,
                m.last_updated_time,
                json.dumps(m.metadata or {}, indent=2),
            ]
            for m in models
        ]

        render.tabulate_or_csv(headers, values, False)


def model_by_name(sess: api.Session, name: str) -> client.Model:
    d = client.Determined._from_session(sess)
    return d.get_model(identifier=name)


def list_versions(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    model = model_by_name(sess, args.name)
    if args.json:
        r = sess.get(f"api/v1/models/{model.model_id}/versions")
        data = r.json()
        render.print_json(data)

    else:
        render_model(model)
        print("\n")
        _render_model_versions(model.list_versions())


def delete(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    model = model_by_name(sess, args.name)
    model.delete()


def create(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    model = d.create_model(args.name, args.description, workspace_name=args.workspace_name)
    if args.json:
        render.print_json(render.model_to_json(model))
    else:
        render_model(model)


def move(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    model = model_by_name(sess, args.name)
    model.move_to_workspace(args.workspace_name)


def describe(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    model = model_by_name(sess, args.name)
    model_version = model.get_version(args.version)

    if args.json:
        render.print_json(render.model_to_json(model))
    else:
        render_model(model)
        if model_version is not None:
            print("\n")
            _render_model_versions([model_version])


def register_version(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    model = model_by_name(sess, args.name)
    if args.json:
        resp = sess.post(
            f"/api/v1/models/{model.model_id}/versions",
            json={"checkpointUuid": args.uuid},
        )

        render.print_json(resp.json())
    else:
        model_version = model.register_version(args.uuid)
        render_model(model)
        print("\n")
        _render_model_versions([model_version])


args_description = [
    cli.Cmd(
        "m|odel",
        None,
        "manage models",
        [
            cli.Cmd(
                "list ls",
                list_models,
                "list all models in the registry",
                [
                    cli.Arg(
                        "-w",
                        "--workspace-names",
                        type=str,
                        help="list models in given list of comma-separated workspaces",
                    ),
                    cli.Arg(
                        "--sort-by",
                        type=str,
                        choices=["name", "description", "creation_time", "last_updated_time"],
                        default="last_updated_time",
                        help="sort models by the given field",
                    ),
                    cli.Arg(
                        "--order-by",
                        type=str,
                        choices=["asc", "desc"],
                        default="asc",
                        help="order models in either ascending or descending order",
                    ),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
                is_default=True,
            ),
            cli.Cmd(
                "register-version",
                register_version,
                "register a new version of a model",
                [
                    cli.Arg("name", type=str, help="name of the model"),
                    cli.Arg(
                        "uuid", type=str, help="uuid to register as the next version of the model"
                    ),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "describe",
                describe,
                "describe model",
                [
                    cli.Arg("name", type=str, help="name of model to describe"),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                    cli.Arg(
                        "--version",
                        type=int,
                        default=-1,
                        help="model version information to include in output",
                    ),
                ],
            ),
            cli.Cmd(
                "list-versions",
                list_versions,
                "list the versions of a model",
                [
                    cli.Arg("name", type=str, help="unique name of the model"),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "delete",
                delete,
                "delete model",
                [
                    cli.Arg("name", type=str, help="unique name of the model"),
                ],
            ),
            cli.Cmd(
                "create",
                create,
                "create model",
                [
                    cli.Arg("name", type=str, help="unique name of the model"),
                    workspace.workspace_arg,
                    cli.Arg("--description", type=str, help="description of the model"),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "move",
                move,
                "move model to given workspace",
                [
                    cli.Arg("name", type=str, help="name of model"),
                    workspace.workspace_arg,
                ],
            ),
        ],
    )
]  # type: List[Any]
