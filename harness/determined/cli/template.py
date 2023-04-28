from argparse import FileType, Namespace
from collections import namedtuple
from typing import Any, Dict, List

from termcolor import colored

from determined import cli
from determined.common import util, yaml
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd

from . import render

TemplateClean = namedtuple("TemplateClean", ["name"])
TemplateAll = namedtuple("TemplateAll", ["name", "config"])


def _parse_config(data: Dict[str, Any]) -> Any:
    # Pretty print the config field.
    return yaml.safe_dump(data, default_flow_style=False)


@authentication.required
def list_template(args: Namespace) -> None:
    templates: List[TemplateAll] = []
    for tpl in bindings.get_GetTemplates(cli.setup_session(args)).templates:
        templates.append(TemplateAll(tpl.name, _parse_config(tpl.config)))
    if args.details:
        render.render_objects(TemplateAll, templates, table_fmt="grid")
    else:
        render.render_objects(TemplateClean, templates)


@authentication.required
def describe_template(args: Namespace) -> None:
    tpl = bindings.get_GetTemplate(
        cli.setup_session(args), templateName=args.template_name
    ).template
    print(_parse_config(tpl.config))


@authentication.required
def set_template(args: Namespace) -> None:
    with args.template_file:
        body = util.safe_load_yaml_with_exceptions(args.template_file)
        v1_template = bindings.v1Template(name=args.template_name, config=body)
        bindings.put_PutTemplate(
            cli.setup_session(args), template_name=args.template_name, body=v1_template
        )
        print(colored("Set template {}".format(args.template_name), "green"))


@authentication.required
def remove_templates(args: Namespace) -> None:
    bindings.delete_DeleteTemplate(cli.setup_session(args), templateName=args.template_name)
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
