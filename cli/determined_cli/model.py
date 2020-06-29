import json
from argparse import Namespace
from typing import Any, List

from determined_common.experimental import Determined, Model, ModelOrderBy, ModelSortBy

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


def create(args: Namespace) -> None:
    model = Determined(args.master, None).create_model(args.name, args.description)

    if args.json:
        print(json.dumps(model.to_json(), indent=2))
    else:
        render_model(model)


def describe(args: Namespace) -> None:
    model = Determined(args.master, None).get_model(args.name)

    if args.json:
        print(json.dumps(model.to_json(), indent=2))
    else:
        render_model(model)


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
                "describe",
                describe,
                "describe model",
                [
                    Arg("name", type=str, help="model to describe"),
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
