"""Shared CLI interface for NTSCs (notebooks, tensorboards, shells, and commands)."""

import argparse
import base64
import collections
import functools
import json
import operator
import os
import pathlib
import re
import urllib
from typing import IO, Any, Dict, Iterable, List, Optional, Tuple, Union

import termcolor

from determined import cli, util
from determined.cli import render, workspace
from determined.common import api, context
from determined.common import util as common_util

CONFIG_DESC = """
Additional configuration arguments for setting up a command.
Arguments should be specified as `key=value`. Nested configuration
keys can be specified by dot notation, e.g., `resources.slots=4`.
List values can be specified by comma-separated values. More
complex configuration values can be specified using JSON, e.g.,
`bind_mounts=[{host_path: "/tmp", container_path: "/tmp"}]`.
"""

CONTEXT_DESC = """
A directory whose contents should be copied into the task container.
Unlike --include, the directory itself will not appear in the task
container, only its contents.  The total bytes copied into the container
must not exceed 96 MB.  By default, no files are copied.  See also:
--include, which preserves the root directory name.
"""

INCLUDE_DESC = """
A file or directory to copy into the task container.  May be provided more
than once.   Unlike --context, --include will preserve the top-level
directory name during the copy.  The total bytes copied into the
container must not exceed 96 MB.  By default, no files are copied.  See
also: --context, when a task should run inside a directory.
"""

VOLUME_DESC = """
A mount specification in the form of `<host path>:<container path>`. The
given path on the host machine will be mounted under the given path in
the command container.
"""


_CONFIG_PATHS_COERCE_TO_LIST = {
    "bind_mounts",
}

TASK_ID_REGEX = re.compile(
    r"^(?:[0-9]+\.)?[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", re.IGNORECASE
)

CommandTableHeader = collections.OrderedDict(
    [
        ("id", "id"),
        ("username", "username"),
        ("description", "description"),
        ("state", "state"),
        ("exitStatus", "exitStatus"),
        ("resourcePool", "resourcePool"),
        ("workspaceName", "workspaceName"),
    ]
)

TensorboardTableHeader = collections.OrderedDict(
    [
        ("id", "id"),
        ("username", "username"),
        ("description", "description"),
        ("state", "state"),
        ("experimentIds", "experimentIds"),
        ("trialIds", "trialIds"),
        ("exitStatus", "exitStatus"),
        ("resourcePool", "resourcePool"),
        ("workspaceName", "workspaceName"),
    ]
)

TaskTypeNotebook = "notebook"
TaskTypeCommand = "command cmd"
TaskTypeShell = "shell"
TaskTypeTensorBoard = "tensorboard"

RemoteTaskName = {
    TaskTypeNotebook: "notebook",
    TaskTypeCommand: "command",
    TaskTypeShell: "shell",
    TaskTypeTensorBoard: "tensorboard",
}

RemoteTaskLogName = {
    TaskTypeNotebook: "Notebook",
    TaskTypeCommand: "Command",
    TaskTypeShell: "Shell",
    TaskTypeTensorBoard: "TensorBoard",
}

RemoteTaskNewAPIs = {
    TaskTypeNotebook: "notebooks",
    TaskTypeCommand: "commands",
    TaskTypeShell: "shells",
    TaskTypeTensorBoard: "tensorboards",
}

RemoteTaskOldAPIs = {
    TaskTypeNotebook: "notebooks",
    TaskTypeCommand: "commands",
    TaskTypeShell: "shells",
    TaskTypeTensorBoard: "tensorboard",
}

RemoteTaskListTableHeaders: Dict[str, Dict[str, str]] = {
    "notebook": CommandTableHeader,
    "command cmd": CommandTableHeader,
    "shell": CommandTableHeader,
    "tensorboard": TensorboardTableHeader,
}

RemoteTaskGetIDsFunc = {
    "notebook": lambda args: args.notebook_id,
    "command cmd": lambda args: args.command_id,
    "shell": lambda args: args.shell_id,
    "tensorboard": lambda args: args.tensorboard_id,
}


ls_sort_args: cli.ArgsDescription = [
    cli.Arg(
        "--sort-by",
        type=str,
        help="sort by the given field",
        choices=list(CommandTableHeader.keys()) + ["startTime"],
        default="startTime",
    ),
    cli.Arg(
        "--order-by",
        type=str,
        choices=["asc", "desc"],
        default="asc",
        help="order in either ascending or descending order",
    ),
]


def expand_uuid_prefixes(
    sess: api.Session, args: argparse.Namespace, prefixes: Optional[Union[str, List[str]]] = None
) -> Union[str, List[str]]:
    if prefixes is None:
        prefixes = RemoteTaskGetIDsFunc[args._command](args)  # type: ignore

    was_single = False
    if isinstance(prefixes, str):
        was_single = True
        prefixes = [prefixes]

    # Avoid making a network request if everything is already a full UUID.
    if not all(TASK_ID_REGEX.match(p) for p in prefixes):
        if args._command not in RemoteTaskNewAPIs:
            raise api.errors.BadRequestException(
                f"partial UUIDs not supported for 'det {args._command} {args._subcommand}'"
            )
        api_path = RemoteTaskNewAPIs[args._command]
        api_full_path = "api/v1/{}".format(api_path)
        res = sess.get(api_full_path).json()[api_path]
        all_ids: List[str] = [x["id"] for x in res]

        def expand(prefix: str) -> str:
            if TASK_ID_REGEX.match(prefix):
                return prefix

            # Could do better algorithmically than repeated linear scans, but let's not complicate
            # the code unless it becomes an issue in practice.
            ids = [x for x in all_ids if x.startswith(prefix)]
            if len(ids) > 1:
                raise api.errors.BadRequestException(f"partial UUID '{prefix}' not unique")
            elif len(ids) == 0:
                raise api.errors.BadRequestException(f"partial UUID '{prefix}' not found")
            return ids[0]

        prefixes = [expand(p) for p in prefixes]

    if was_single:
        prefixes = prefixes[0]
    return prefixes


def describe(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    task_id = expand_uuid_prefixes(sess, args)
    item = sess.get(f"api/v1/{RemoteTaskNewAPIs[args._command]}/{task_id}").json()["command"]

    w_names = workspace.get_workspace_names(sess)
    if item["state"].startswith("STATE_"):
        item["state"] = item["state"].replace("STATE_", "")
    if "workspaceId" in item:
        wId = item["workspaceId"]
        item["workspaceName"] = w_names[wId] if wId in w_names else f"missing workspace id {wId}"

    if getattr(args, "json", None):
        render.print_json(item)
        return

    table_header = RemoteTaskListTableHeaders[args._command]
    values = render.select_values([item], table_header)
    render.tabulate_or_csv(table_header, values, getattr(args, "csv", False))


def list_tasks(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    api_path = RemoteTaskNewAPIs[args._command]
    api_full_path = "api/v1/{}".format(api_path)
    table_header = RemoteTaskListTableHeaders[args._command]

    params: Dict[str, Any] = {}

    if "workspace_name" in args and args.workspace_name is not None:
        workspace_obj = api.workspace_by_name(sess, args.workspace_name)

        params["workspaceId"] = workspace_obj.id

    if not args.all:
        params["users"] = [sess.username]

    res = sess.get(api_full_path, params=params).json()[api_path]

    if args.quiet:
        for command in res:
            print(command["id"])
        return

    # swap workspace_id for workspace name.
    w_names = workspace.get_workspace_names(sess)

    for item in res:
        if item["state"].startswith("STATE_"):
            item["state"] = item["state"][6:]
        if "workspaceId" in item:
            wId = item["workspaceId"]
            item["workspaceName"] = (
                w_names[wId] if wId in w_names else f"missing workspace id {wId}"
            )

    res.sort(key=operator.itemgetter(args.sort_by), reverse=args.order_by == "desc")

    if getattr(args, "json", None):
        render.print_json(res)
        return

    values = render.select_values(res, table_header)

    render.tabulate_or_csv(table_header, values, getattr(args, "csv", False))


def kill(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    task_ids = expand_uuid_prefixes(sess, args)
    name = RemoteTaskName[args._command]

    for i, task_id in enumerate(task_ids):
        try:
            _kill(sess, args._command, task_id)
            print(termcolor.colored("Killed {} {}".format(name, task_id), "green"))
        except api.errors.APIException as e:
            if not args.force:
                for ignored in task_ids[i + 1 :]:
                    print("Cowardly not killing {}".format(ignored))
                raise e
            print(termcolor.colored("Skipping: {} ({})".format(e, type(e).__name__), "red"))


def _kill(sess: api.Session, task_type: str, task_id: str) -> None:
    sess.post(f"api/v1/{RemoteTaskNewAPIs[task_type]}/{task_id}/kill")


def set_priority(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    task_id = expand_uuid_prefixes(sess, args)
    name = RemoteTaskName[args._command]

    try:
        api_full_path = f"api/v1/{RemoteTaskNewAPIs[args._command]}/{task_id}/set_priority"
        sess.post(api_full_path, json={"priority": args.priority})
        print(termcolor.colored(f"Set priority of {name} {task_id} to {args.priority}", "green"))
    except api.errors.APIException as e:
        print(termcolor.colored(f"Skipping: {e} ({type(e).__name__})", "red"))


def config(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    task_id = expand_uuid_prefixes(sess, args)
    res_json = sess.get(f"api/v1/{RemoteTaskNewAPIs[args._command]}/{task_id}").json()
    print(render.format_object_as_yaml(res_json["config"]))


# Convert config overrides in dot notation to json path
# For example the caller of this func will break down the config override
#   --config slurm.sbatch_args=[--time=05:00]
# into key slurm.sbatch_args, value [--time=05:00]
# And this func will return string in json form:
# 'slurm': {'sbatch_args': ['--time=05:00']}
def _dot_to_json(key: Any, value: Any) -> Any:
    json_output: Any = {}
    path = key.split(".")
    target = functools.reduce(lambda d, k: d.setdefault(k, {}), path[:-1], json_output)
    target[path[-1]] = value
    return json_output


# A recursive function to replace value val in a Dict with a new value rval for specified keys.
# Some key may have value None of NoneType when value is not defined.
# For example, when parent key "slurm" is specified in experiment .yaml file without
# any child, the experiment config will contain None value for key 'slurm':
# {'name': 'noop_single', ..., 'entrypoint': 'model_def:NoOpTrial', 'slurm': None}
# This NoneType value None will raise TypeError if used, for example:
# TypeError: argument of type 'NoneType' is not iterable
# TypeError: 'NoneType' object does not support item assignment
# The purpose of this function is to convert None value to empty value
# but we only do this conversion for specified keys.
def _replace_value(data_dict: Dict[str, Any], keys: List[str], val: Any, rval: Any) -> None:
    for key in data_dict.keys():
        if data_dict[key] == val and key in keys:
            data_dict[key] = rval
        elif type(data_dict[key]) is dict:
            _replace_value(data_dict[key], keys, val, rval)


def parse_config_overrides(config: Dict[str, Any], overrides: Iterable[str]) -> Dict[str, Any]:
    for config_arg in overrides:
        if "=" not in config_arg:
            raise ValueError(
                "Could not read configuration option '{}'\n\n"
                "Expecting:\n{}".format(config_arg, CONFIG_DESC)
            )

        key, value = config_arg.split("=", maxsplit=1)  # type: Tuple[str, Any]

        # Complex objects may contain commas but are not intended to be split
        # on commas and have their parts parsed separately.
        if value.startswith(("[", "{")):
            # Certain configurations keys are expected to have list values.
            # Convert a single value to a singleton list if needed.
            if key in _CONFIG_PATHS_COERCE_TO_LIST and value.startswith("{"):
                value = [common_util.yaml_safe_load(value)]
            else:
                value = common_util.yaml_safe_load(value)
        # Separate values if a comma exists. Use yaml_safe_load() to cast
        # the value(s) to the type YAML would use, e.g., "4" -> 4.
        elif "," in value:
            value = [common_util.yaml_safe_load(v) for v in value.split(",")]
        else:
            value = common_util.yaml_safe_load(value)
            # Certain configurations keys are expected to have list values.
            # Convert a single value to a singleton list if needed.
            if key in _CONFIG_PATHS_COERCE_TO_LIST:
                value = [value]

        # Convert config override in dot notation to json path
        config_arg_in_json = _dot_to_json(key, value)

        # Some key may have value None of NoneType when value is not defined.
        # For example, when parent key "slurm" is specified in experiment .yaml file without
        # any child key, the config will contain None value for key 'slurm':
        # {'name': 'noop_single', ..., 'entrypoint': 'model_def:NoOpTrial', 'slurm': None}
        # This NoneType value None will raise TypeError if used, for example:
        # TypeError: argument of type 'NoneType' is not iterable
        # TypeError: 'NoneType' object does not support item assignment
        # Before we can merge the config from the exp .yaml file with the config_arg
        # provided from the command line, we need to convert NoneType value None in
        # config to empty string {}. We only convert the None value for the key
        # specified in config_arg in overrides, for the example above, if the override
        # is "--config slurm.sbatch_args=[--mem-per-gpu=1g]", we will convert the
        # " 'slurm': None " to " 'slurm': {} " in config before pass it to merge_dicts
        # function.
        _replace_value(config, key.split("."), None, {})
        # Merge two objects in json format
        config = util.merge_dicts(config, config_arg_in_json)

    return config


def parse_config(
    config_file: Optional[IO],
    entrypoint: Optional[List[str]],
    overrides: Iterable[str],
    volumes: Iterable[str],
) -> Dict[str, Any]:
    config = {}  # type: Dict[str, Any]
    if config_file:
        with config_file:
            config = common_util.safe_load_yaml_with_exceptions(config_file)

    config = parse_config_overrides(config, overrides)

    for volume_arg in volumes:
        if ":" not in volume_arg:
            raise ValueError(
                "Could not read volume option '{}'\n\n"
                "Expecting:\n{}".format(volume_arg, VOLUME_DESC)
            )

        host_path, container_path = volume_arg.split(":", maxsplit=1)
        bind_mounts = config.setdefault("bind_mounts", [])
        bind_mounts.append({"host_path": host_path, "container_path": container_path})

    # Use the entrypoint command line argument if an entrypoint has not already been
    # defined by previous settings.
    if not config.get("entrypoint") and entrypoint:
        config["entrypoint"] = entrypoint

    return config


def launch_command(
    sess: api.Session,
    endpoint: str,
    config: Dict[str, Any],
    template: str,
    context_path: Optional[pathlib.Path] = None,
    includes: Iterable[pathlib.Path] = (),
    data: Optional[Dict[str, Any]] = None,
    workspace_id: Optional[int] = None,
    preview: Optional[bool] = False,
    default_body: Optional[Dict[str, Any]] = None,
) -> Any:
    user_files = context.read_legacy_context(context_path, includes)

    body = {}  # type: Dict[str, Any]
    if default_body:
        body.update(default_body)

    body["config"] = config

    if template:
        body["template_name"] = template

    if len(user_files) > 0:
        body["files"] = user_files

    if data is not None:
        message_bytes = json.dumps(data).encode("utf-8")
        base64_bytes = base64.b64encode(message_bytes)
        body["data"] = base64_bytes

    if preview:
        body["preview"] = preview

    if workspace_id is not None:
        body["workspaceId"] = workspace_id

    return sess.post(endpoint, json=body).json()


def make_interactive_task_url(
    task_id: str,
    service_address: str,
    description: str,
    resource_pool: str,
    task_type: str,
    currentSlotsExceeded: bool,
) -> str:
    wait_path = (
        "/jupyter-lab/{}/events".format(task_id)
        if task_type == "jupyter-lab"
        else "/tensorboard/{}/events?tail=1".format(task_id)
    )
    wait_path_url = service_address + wait_path
    public_url = os.environ.get("PUBLIC_URL", "/det")
    wait_page_url = "{}/wait/{}/{}?eventUrl={}&serviceAddr={}".format(
        public_url, task_type, task_id, wait_path_url, service_address
    )
    task_web_url = "{}/interactive/{}/{}/{}/{}/{}?{}".format(
        public_url,
        task_id,
        task_type,
        urllib.parse.quote(description),
        resource_pool,
        urllib.parse.quote_plus(wait_page_url),
        f"currentSlotsExceeded={str(currentSlotsExceeded).lower()}",
    )
    # Return a relative path that can be joined to the master_url with a simple "/" separator.
    return task_web_url.lstrip("/")
