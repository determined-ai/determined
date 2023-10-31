import json
from argparse import Namespace
from typing import Any, List

import determined.cli.render
from determined import cli
from determined.cli import render
from determined.common import api
from determined.common.api import authentication
from determined.common.declarative_argparse import Arg, Cmd
from determined.experimental import Determined, Model, ModelSortBy, ModelVersion, OrderBy


def render_model(model: Model) -> None:
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


def _render_model_versions(model_versions: List[ModelVersion]) -> None:
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


def list_models(args: Namespace) -> None:
    workspace_names = None
    if args.workspace_names is not None:
        workspace_names = args.workspace_names.split(",")
    models = Determined(args.master, args.user).list_models(
        sort_by=ModelSortBy[args.sort_by.upper()],
        order_by=OrderBy[args.order_by.upper()],
        workspace_names=workspace_names,
    )
    if args.json:
        determined.cli.render.print_json([render.model_to_json(m) for m in models])
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


def model_by_name(args: Namespace) -> Model:
    return Determined(args.master, args.user).get_model(identifier=args.name)


@authentication.required
def list_versions(args: Namespace) -> None:
    model = model_by_name(args)
    if args.json:
        r = api.get(args.master, "api/v1/models/{}/versions".format(model.model_id))
        data = r.json()
        determined.cli.render.print_json(data)

    else:
        render_model(model)
        print("\n")
        _render_model_versions(model.list_versions())


def create(args: Namespace) -> None:
    model = Determined(args.master, args.user).create_model(
        args.name, args.description, workspace_name=args.workspace_name
    )

    if args.json:
        determined.cli.render.print_json(render.model_to_json(model))
    else:
        render_model(model)


def move(args: Namespace) -> None:
    model = model_by_name(args)
    model.move_to_workspace(args.workspace_name)


def describe(args: Namespace) -> None:
    model = model_by_name(args)
    model_version = model.get_version(args.version)

    if args.json:
        determined.cli.render.print_json(render.model_to_json(model))
    else:
        render_model(model)
        if model_version is not None:
            print("\n")
            _render_model_versions([model_version])


@authentication.required
def register_version(args: Namespace) -> None:
    model = model_by_name(args)
    if args.json:
        resp = api.post(
            args.master,
            "/api/v1/models/{}/versions".format(model.model_id),
            json={"checkpointUuid": args.uuid},
        )

        determined.cli.render.print_json(resp.json())
    else:
        model_version = model.register_version(args.uuid)
        render_model(model)
        print("\n")
        _render_model_versions([model_version])


args_description = [
    Cmd(
        "m|odel",
        None,
        "manage models",
        [
            Cmd(
                "list ls",
                list_models,
                "list all models in the registry",
                [
                    Arg(
                        "-w",
                        "--workspace-names",
                        type=str,
                        help="list models in given list of comma-separated workspaces",
                    ),
                    Arg(
                        "--sort-by",
                        type=str,
                        choices=["name", "description", "creation_time", "last_updated_time"],
                        default="last_updated_time",
                        help="sort models by the given field",
                    ),
                    Arg(
                        "--order-by",
                        type=str,
                        choices=["asc", "desc"],
                        default="asc",
                        help="order models in either ascending or descending order",
                    ),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
                is_default=True,
            ),
            Cmd(
                "register-version",
                register_version,
                "register a new version of a model",
                [
                    Arg("name", type=str, help="name of the model"),
                    Arg("uuid", type=str, help="uuid to register as the next version of the model"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "describe",
                describe,
                "describe model",
                [
                    Arg("name", type=str, help="name of model to describe"),
                    Arg("--json", action="store_true", help="print as JSON"),
                    Arg(
                        "--version",
                        type=int,
                        default=-1,
                        help="model version information to include in output",
                    ),
                ],
            ),
            Cmd(
                "list-versions",
                list_versions,
                "list the versions of a model",
                [
                    Arg("name", type=str, help="unique name of the model"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "create",
                create,
                "create model",
                [
                    Arg("name", type=str, help="unique name of the model"),
                    cli.workspace.workspace_arg,
                    Arg("--description", type=str, help="description of the model"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "move",
                move,
                "move model to given workspace",
                [
                    Arg("name", type=str, help="name of model"),
                    cli.workspace.workspace_arg,
                ],
            ),
        ],
    )
]  # type: List[Any]
