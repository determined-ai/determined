import json
from argparse import Namespace
from typing import Any, List

from determined_common import api
from determined_common.api.authentication import authentication_required
from determined_common.experimental import Checkpoint, Determined, Model, ModelOrderBy, ModelSortBy

from . import render
from .declarative_argparse import Arg, Cmd


def render_model(model: Model) -> None:
    table = [
        ["Name", model.name],
        ["Description", model.description],
        ["Creation Time", model.creation_time],
        ["Last Updated Time", model.last_updated_time],
        ["Metadata", json.dumps(model.metadata or {}, indent=4)],
    ]

    headers, values = zip(*table)  # type: ignore

    render.tabulate_or_csv(headers, [values], False)


def render_model_version(checkpoint: Checkpoint) -> None:
    headers = [
        "Version #",
        "Trial ID",
        "Batch #",
        "Checkpoint UUID",
        "Validation Metrics",
        "Metadata",
    ]

    values = [
        [
            checkpoint.model_version,
            checkpoint.trial_id,
            checkpoint.batch_number,
            checkpoint.uuid,
            json.dumps(checkpoint.validation, indent=2),
            json.dumps(checkpoint.metadata, indent=2),
        ]
    ]

    render.tabulate_or_csv(headers, values, False)


def list_models(args: Namespace) -> None:
    models = Determined(args.master, None).get_models(
        sort_by=ModelSortBy[args.sort_by.upper()], order_by=ModelOrderBy[args.order_by.upper()]
    )
    if args.json:
        print(json.dumps([m.to_json() for m in models], indent=2))
    else:
        headers = ["Name", "Creation Time", "Last Updated Time", "Metadata"]

        values = [
            [m.name, m.creation_time, m.last_updated_time, json.dumps(m.metadata or {}, indent=2)]
            for m in models
        ]

        render.tabulate_or_csv(headers, values, False)


@authentication_required
def list_versions(args: Namespace) -> None:
    if args.json:
        r = api.get(args.master, "models/{}/versions".format(args.name))
        data = r.json()
        print(json.dumps(data, indent=2))

    else:
        model = Determined(args.master).get_model(args.name)
        render_model(model)
        print("\n")

        headers = [
            "Version #",
            "Trial ID",
            "Batch #",
            "Checkpoint UUID",
            "Validation Metrics",
            "Metadata",
        ]

        values = [
            [
                ckpt.model_version,
                ckpt.trial_id,
                ckpt.batch_number,
                ckpt.uuid,
                json.dumps(ckpt.validation, indent=2),
                json.dumps(ckpt.metadata, indent=2),
            ]
            for ckpt in model.get_versions()
        ]

        render.tabulate_or_csv(headers, values, False)


def create(args: Namespace) -> None:
    model = Determined(args.master, None).create_model(args.name, args.description)

    if args.json:
        print(json.dumps(model.to_json(), indent=2))
    else:
        render_model(model)


def describe(args: Namespace) -> None:
    model = Determined(args.master, None).get_model(args.name)
    checkpoint = model.get_version(args.version)

    if args.json:
        print(json.dumps(model.to_json(), indent=2))
    else:
        render_model(model)
        if checkpoint is not None:
            print("\n")
            render_model_version(checkpoint)


@authentication_required
def register_version(args: Namespace) -> None:
    if args.json:
        resp = api.post(
            args.master,
            "/api/v1/models/{}/versions".format(args.name),
            body={"checkpoint_uuid": args.uuid},
        )

        print(json.dumps(resp.json(), indent=2))
    else:
        model = Determined(args.master, None).get_model(args.name)
        checkpoint = model.register_version(args.uuid)
        render_model(model)
        print("\n")
        render_model_version(checkpoint)


args_description = [
    Cmd(
        "m|odel",
        None,
        "manage models",
        [
            Cmd(
                "list",
                list_models,
                "list all models in the registry",
                [
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
                    Arg("name", type=str, help="model to describe"),
                    Arg("--json", action="store_true", help="print as JSON"),
                    Arg(
                        "--version",
                        type=int,
                        default=0,
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
                    Arg("--description", type=str, help="description of the model"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
        ],
    )
]  # type: List[Any]
