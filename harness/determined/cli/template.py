from argparse import ArgumentError, FileType, Namespace
from collections import namedtuple
from typing import Any, Dict, List

from termcolor import colored

from determined import cli
from determined.cli import render
from determined.cli.workspace import get_workspace_id_from_args, workspace_arg
from determined.common import api, util
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd

TemplateClean = namedtuple("TemplateClean", ["name", "workspace"])
TemplateAll = namedtuple("TemplateAll", ["name", "workspace", "config"])


def _parse_config(data: Dict[str, Any]) -> Any:
    # Pretty print the config field.
    return util.yaml_safe_dump(data, default_flow_style=False)


@authentication.required
def list_template(args: Namespace) -> None:
    templates: List[TemplateAll] = []
    w_names = cli.workspace.get_workspace_names(cli.setup_session(args))

    for tpl in bindings.get_GetTemplates(cli.setup_session(args)).templates:
        w_name = w_names.get(tpl.workspaceId, "missing workspace")
        templates.append(TemplateAll(tpl.name, w_name, _parse_config(tpl.config)))
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
        """
        WARN: this downgrades the atomic behavior of upsert but it's
        an acceptable tradeoff for now until we can remove this command.
        """
        session = cli.setup_session(args)
        body = util.safe_load_yaml_with_exceptions(args.template_file)
        try:
            bindings.get_GetTemplate(session, templateName=args.template_name).template
            bindings.patch_PatchTemplateConfig(session, templateName=args.template_name, body=body)
        except api.errors.NotFoundException:
            v1_template = bindings.v1Template(name=args.template_name, config=body, workspaceId=0)
            bindings.post_PostTemplate(session, template_name=args.template_name, body=v1_template)
        print(colored("Set template {}".format(args.template_name), "green"))


@authentication.required
def create_template(args: Namespace) -> None:
    if not args.template_file:
        raise ArgumentError(None, "template_file is required for set command")
    body = util.safe_load_yaml_with_exceptions(args.template_file)
    workspace_id = get_workspace_id_from_args(args) or 0
    v1_template = bindings.v1Template(
        name=args.template_name, config=body, workspaceId=workspace_id
    )
    bindings.post_PostTemplate(
        cli.setup_session(args), template_name=args.template_name, body=v1_template
    )
    print(colored("Created template {}".format(args.template_name), "green"))


@authentication.required
def patch_template_config(args: Namespace) -> None:
    if not args.template_file:
        raise ArgumentError(None, "template_file is required for set command")
    body = util.safe_load_yaml_with_exceptions(args.template_file)
    bindings.patch_PatchTemplateConfig(
        cli.setup_session(args), templateName=args.template_name, body=body
    )
    print(colored("Updated template {}".format(args.template_name), "green"))


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
        Cmd("set", set_template, "set config template", [
            Arg("template_name", help="template name"),
            Arg("template_file", type=FileType("r"),
                help="config template file (.yaml)"),
        ], deprecation_message="use the following options `det template create|set-value`."),
        Cmd("set-value", None, "set template attributes", [
            Cmd("config", patch_template_config, "update config template", [
                Arg("template_name", help="template name"),
                Arg("template_file", type=FileType("r"),
                    help="config template file (.yaml)"),
            ]),
        ]),
        Cmd("describe", describe_template,
            "describe config template", [
                Arg("template_name", type=str, help="template name"),
            ]),
        Cmd("create", create_template, "create config template", [
            Arg("template_name", help="template name"),
            Arg("template_file", type=FileType("r"),
                help="config template file (.yaml)"),
            workspace_arg,
        ]),
        Cmd("remove rm", remove_templates,
            "remove config template", [
                Arg("template_name", help="template name")
            ]),
    ])
]  # type: List[Any]

# fmt: on
