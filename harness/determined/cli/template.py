import argparse
import collections
from typing import Any, Dict, List

import termcolor

from determined import cli
from determined.cli import render, workspace
from determined.common import api, util
from determined.common.api import bindings

TemplateClean = collections.namedtuple("TemplateClean", ["name", "workspace"])
TemplateAll = collections.namedtuple("TemplateAll", ["name", "workspace", "config"])


def _parse_config(data: Dict[str, Any]) -> Any:
    # Pretty print the config field.
    return util.yaml_safe_dump(data, default_flow_style=False)


def list_template(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    templates: List[TemplateAll] = []
    w_names = workspace.get_workspace_names(sess)

    for tpl in bindings.get_GetTemplates(sess).templates:
        w_name = w_names.get(tpl.workspaceId, "missing workspace")
        templates.append(TemplateAll(tpl.name, w_name, _parse_config(tpl.config)))
    if args.details:
        render.render_objects(TemplateAll, templates, table_fmt="grid")
    else:
        render.render_objects(TemplateClean, templates)


def describe_template(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    tpl = bindings.get_GetTemplate(sess, templateName=args.template_name).template
    print(_parse_config(tpl.config))


def set_template(args: argparse.Namespace) -> None:
    with args.template_file:
        """
        WARN: this downgrades the atomic behavior of upsert but it's
        an acceptable tradeoff for now until we can remove this command.
        """
        sess = cli.setup_session(args)
        body = util.safe_load_yaml_with_exceptions(args.template_file)
        try:
            bindings.get_GetTemplate(sess, templateName=args.template_name).template
            bindings.patch_PatchTemplateConfig(sess, templateName=args.template_name, body=body)
        except api.errors.NotFoundException:
            v1_template = bindings.v1Template(name=args.template_name, config=body, workspaceId=0)
            bindings.post_PostTemplate(sess, template_name=args.template_name, body=v1_template)
        print(termcolor.colored("Set template {}".format(args.template_name), "green"))


def create_template(args: argparse.Namespace) -> None:
    if not args.template_file:
        raise argparse.ArgumentError(None, "template_file is required for set command")
    sess = cli.setup_session(args)
    body = util.safe_load_yaml_with_exceptions(args.template_file)
    workspace_id = workspace.get_workspace_id_from_args(args) or 0
    v1_template = bindings.v1Template(
        name=args.template_name, config=body, workspaceId=workspace_id
    )
    bindings.post_PostTemplate(sess, template_name=args.template_name, body=v1_template)
    print(termcolor.colored("Created template {}".format(args.template_name), "green"))


def patch_template_config(args: argparse.Namespace) -> None:
    if not args.template_file:
        raise argparse.ArgumentError(None, "template_file is required for set command")
    sess = cli.setup_session(args)
    body = util.safe_load_yaml_with_exceptions(args.template_file)
    bindings.patch_PatchTemplateConfig(sess, templateName=args.template_name, body=body)
    print(termcolor.colored("Updated template {}".format(args.template_name), "green"))


def remove_templates(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    bindings.delete_DeleteTemplate(sess, templateName=args.template_name)
    print(termcolor.colored("Removed template {}".format(args.template_name), "green"))


# fmt: off

args_description = [
    cli.Cmd("template tpl", None, "manage config templates", [
        cli.Cmd("list ls", list_template, "list config templates", [
            cli.Arg("-d", "--details", action="store_true",
                help="show the configs of the templates"),
        ], is_default=True),
        cli.Cmd("set", set_template, "set config template", [
            cli.Arg("template_name", help="template name"),
            cli.Arg("template_file", type=argparse.FileType("r"),
                help="config template file (.yaml)"),
        ], deprecation_message="use the following options `det template create|set-value`."),
        cli.Cmd("set-value", None, "set template attributes", [
            cli.Cmd("config", patch_template_config, "update config template", [
                cli.Arg("template_name", help="template name"),
                cli.Arg("template_file", type=argparse.FileType("r"),
                    help="config template file (.yaml)"),
            ]),
        ]),
        cli.Cmd(
            "describe", describe_template,
            "describe config template", [
                cli.Arg("template_name", type=str, help="template name"),
            ]
        ),
        cli.Cmd("create", create_template, "create config template", [
            cli.Arg("template_name", help="template name"),
            cli.Arg("template_file", type=argparse.FileType("r"),
                help="config template file (.yaml)"),
            workspace.workspace_arg,
        ]),
        cli.Cmd(
            "remove rm", remove_templates,
            "remove config template", [
                cli.Arg("template_name", help="template name")
            ]
        ),
    ])
]  # type: List[Any]

# fmt: on
