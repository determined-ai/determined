from argparse import FileType, Namespace
from collections import namedtuple
from typing import Any, List

from ruamel import yaml
from termcolor import colored

from determined_common import api

from . import render
from .declarative_argparse import Arg, Cmd
from .user import authentication_required

TemplateClean = namedtuple("TemplateClean", ["name"])
TemplateAll = namedtuple("TemplateAll", ["name", "config"])


@authentication_required
def list_template(args: Namespace) -> None:
    q = api.GraphQLQuery(args.master)
    tmpl = q.op.templates()
    tmpl.name()
    if args.details:
        tmpl.config()
    resp = q.send()
    print(resp)

    if args.details:
        res_format = [
            {"name": item.name, "config": yaml.safe_dump(item.config, default_flow_style=False)}
            for item in resp.templates
        ]
        render.render_dicts(TemplateAll, res_format, table_fmt="grid")
    else:
        render.render_dicts(TemplateClean, resp.templates)


@authentication_required
def describe_template(args: Namespace) -> None:
    q = api.GraphQLQuery(args.master)
    q.op.templates_by_pk(name=args.template_name).config()
    resp = q.send()
    print(yaml.safe_dump(resp.templates_by_pk.config, default_flow_style=False))


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
