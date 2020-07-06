import base64
from argparse import FileType, Namespace
from collections import namedtuple
from typing import Any, List

from ruamel import yaml
from termcolor import colored

from determined_common import api
from determined_common.api.authentication import authentication_required

from . import render
from .declarative_argparse import Arg, Cmd

TemplateClean = namedtuple("TemplateClean", ["name"])
TemplateAll = namedtuple("TemplateAll", ["name", "config"])


def _parse_config(field: Any) -> Any:
    # Pretty print the config field.
    return yaml.safe_dump(yaml.safe_load(base64.b64decode(field)), default_flow_style=False)


@authentication_required
def list_template(args: Namespace) -> None:
    templates = [
        render.unmarshal(TemplateAll, t, {"config": _parse_config})
        for t in api.get(args.master, path="templates").json()
    ]
    if args.details:
        render.render_objects(TemplateAll, templates, table_fmt="grid")
    else:
        render.render_objects(TemplateClean, templates)


@authentication_required
def describe_template(args: Namespace) -> None:
    resp = api.get(args.master, path="templates/{}".format(args.template_name)).json()
    template = render.unmarshal(TemplateAll, resp, {"config": _parse_config})
    print(template.config)


@authentication_required
def set_template(args: Namespace) -> None:
    with args.template_file:
        body = yaml.safe_load(args.template_file)
        api.put(args.master, path="templates/" + args.template_name, body=body)
        print(colored("Set template {}".format(args.template_name), "green"))


@authentication_required
def remove_templates(args: Namespace) -> None:
    api.delete(args.master, path="templates/" + args.template_name)
    print(colored("Removed template {}".format(args.template_name), "green"))


# fmt: off

args_description = [
    Cmd("template tpl", None, "manage config templates", [
        Cmd("list ls", list_template, "list config templates", [
            Arg("-d", "--details", action="store_true",
                help="show the configs of the templates"),
        ], is_default=True),
        Cmd("describe", describe_template,
            "describe config template", [
                Arg("template_name", type=str, help="template name"),
            ]),
        Cmd("set", set_template, "set config template", [
            Arg("template_name", help="template name"),
            Arg("template_file", type=FileType("r"),
                help="config template file (.yaml)")
        ]),
        Cmd("remove rm", remove_templates,
            "remove config template", [
                Arg("template_name", help="template name")
            ]),
    ])
]  # type: List[Any]

# fmt: on
