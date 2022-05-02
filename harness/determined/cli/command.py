import base64
import json
import re
from argparse import Namespace
from collections import OrderedDict, namedtuple
from pathlib import Path
from typing import IO, Any, Dict, Iterable, List, Optional, Tuple, Union

from termcolor import colored

from determined.cli import render
from determined.common import api, context, util, yaml
from determined.common.api import authentication

CONFIG_DESC = """
Additional configuration arguments for setting up a command.
Arguments should be specified as `key=value`. Nested configuration
keys can be specified by dot notation, e.g., `resources.slots=4`.
List values can be specified by comma-separated values.
"""

CONTEXT_DESC = """
The filepath to a directory that contains the set of files used to
execute the command. All files under this directory will be packaged,
maintaining the existing directory structure. The total byte contents
of the directory must not exceed 96 MB. By default, the context
directory will be empty.
"""

VOLUME_DESC = """
A mount specification in the form of `<host path>:<container path>`. The
given path on the host machine will be mounted under the given path in
the command container.
"""


_CONFIG_PATHS_COERCE_TO_LIST = {
    "bind_mounts",
}

UUID_REGEX = re.compile(
    "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", re.IGNORECASE
)

CommandTableHeader = OrderedDict(
    [
        ("id", "id"),
        ("username", "username"),
        ("description", "description"),
        ("state", "state"),
        ("exitStatus", "exitStatus"),
        ("resourcePool", "resourcePool"),
    ]
)

TensorboardTableHeader = OrderedDict(
    [
        ("id", "id"),
        ("username", "username"),
        ("description", "description"),
        ("state", "state"),
        ("experimentIds", "experimentIds"),
        ("trialIds", "trialIds"),
        ("exitStatus", "exitStatus"),
        ("resourcePool", "resourcePool"),
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

RemoteTaskListTableHeaders = {
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


Command = namedtuple(
    "Command",
    [
        "id",
        "owner",
        "registered_time",
        "config",
        "state",
        "addresses",
        "exit_status",
        "misc",
        "agent_user_group",
    ],
)


def expand_uuid_prefixes(
    args: Namespace, prefixes: Optional[Union[str, List[str]]] = None
) -> Union[str, List[str]]:
    if prefixes is None:
        prefixes = RemoteTaskGetIDsFunc[args._command](args)  # type: ignore

    was_single = False
    if isinstance(prefixes, str):
        was_single = True
        prefixes = [prefixes]

    # Avoid making a network request if everything is already a full UUID.
    if not all(UUID_REGEX.match(p) for p in prefixes):
        api_path = RemoteTaskNewAPIs[args._command]
        api_full_path = "api/v1/{}".format(api_path)
        res = api.get(args.master, api_full_path).json()[api_path]
        all_ids: List[str] = [x["id"] for x in res]

        def expand(prefix: str) -> str:
            if UUID_REGEX.match(prefix):
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


@authentication.required
def list_tasks(args: Namespace) -> None:
    api_path = RemoteTaskNewAPIs[args._command]
    api_full_path = "api/v1/{}".format(api_path)
    table_header = RemoteTaskListTableHeaders[args._command]

    if args.all:
        params = {}  # type: Dict[str, Any]
    else:
        params = {"users": [authentication.must_cli_auth().get_session_user()]}

    res = api.get(args.master, api_full_path, params=params).json()[api_path]

    if args.quiet:
        for command in res:
            print(command["id"])
        return

    for item in res:
        if item["state"].startswith("STATE_"):
            item["state"] = item["state"][6:]

    if getattr(args, "json", None):
        print(json.dumps(res, indent=4))
        return

    values = render.select_values(res, table_header)

    render.tabulate_or_csv(table_header, values, getattr(args, "csv", False))


@authentication.required
def kill(args: Namespace) -> None:
    task_ids = expand_uuid_prefixes(args)
    name = RemoteTaskName[args._command]

    for i, task_id in enumerate(task_ids):
        try:
            _kill(args.master, args._command, task_id)
            print(colored("Killed {} {}".format(name, task_id), "green"))
        except api.errors.APIException as e:
            if not args.force:
                for ignored in task_ids[i + 1 :]:
                    print("Cowardly not killing {}".format(ignored))
                raise e
            print(colored("Skipping: {} ({})".format(e, type(e).__name__), "red"))


def _kill(master_url: str, taskType: str, taskID: str) -> None:
    api_full_path = "api/v1/{}/{}/kill".format(RemoteTaskNewAPIs[taskType], taskID)
    api.post(master_url, api_full_path)


@authentication.required
def set_priority(args: Namespace) -> None:
    task_id = expand_uuid_prefixes(args)
    name = RemoteTaskName[args._command]

    try:
        api_full_path = "api/v1/{}/{}/set_priority".format(
            RemoteTaskNewAPIs[args._command], task_id
        )
        api.post(args.master, api_full_path, {"priority": args.priority})
        print(colored("Set priority of {} {} to {}".format(name, task_id, args.priority), "green"))
    except api.errors.APIException as e:
        print(colored("Skipping: {} ({})".format(e, type(e).__name__), "red"))


@authentication.required
def config(args: Namespace) -> None:
    task_id = expand_uuid_prefixes(args)
    api_full_path = "api/v1/{}/{}".format(RemoteTaskNewAPIs[args._command], task_id)
    res_json = api.get(args.master, api_full_path).json()
    print(render.format_object_as_yaml(res_json["config"]))


def _set_nested_config(config: Dict[str, Any], key_path: List[str], value: Any) -> Dict[str, Any]:
    current = config
    for key in key_path[:-1]:
        current = current.setdefault(key, {})
    current[key_path[-1]] = value
    return config


def parse_config_overrides(config: Dict[str, Any], overrides: Iterable[str]) -> None:
    for config_arg in overrides:
        if "=" not in config_arg:
            raise ValueError(
                "Could not read configuration option '{}'\n\n"
                "Expecting:\n{}".format(config_arg, CONFIG_DESC)
            )

        key, value = config_arg.split("=", maxsplit=1)  # type: Tuple[str, Any]

        # Separate values if a comma exists. Use yaml.safe_load() to cast
        # the value(s) to the type YAML would use, e.g., "4" -> 4.
        if "," in value:
            value = [yaml.safe_load(v) for v in value.split(",")]
        else:
            value = yaml.safe_load(value)

            # Certain configurations keys are expected to have list values.
            # Convert a single value to a singleton list if needed.
            if key in _CONFIG_PATHS_COERCE_TO_LIST:
                value = [value]

        # TODO(#2703): Consider using full JSONPath spec instead of dot
        # notation.
        config = _set_nested_config(config, key.split("."), value)


def parse_config(
    config_file: Optional[IO],
    entrypoint: Optional[List[str]],
    overrides: Iterable[str],
    volumes: Iterable[str],
) -> Dict[str, Any]:
    config = {}  # type: Dict[str, Any]
    if config_file:
        with config_file:
            config = util.safe_load_yaml_with_exceptions(config_file)

    parse_config_overrides(config, overrides)

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
    master: str,
    endpoint: str,
    config: Dict[str, Any],
    template: str,
    context_path: Optional[Path] = None,
    data: Optional[Dict[str, Any]] = None,
    preview: Optional[bool] = False,
    default_body: Optional[Dict[str, Any]] = None,
) -> Any:
    user_files = []  # type: List[Dict[str, Any]]
    if context_path:
        user_files, _ = context.read_context(context_path)

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

    return api.post(
        master,
        endpoint,
        body,
    ).json()


def render_event_stream(event: Any) -> None:
    description = event["description"]
    if event["scheduled_event"] is not None:
        print(
            colored("Scheduling {} (id: {})...".format(description, event["parent_id"]), "yellow")
        )
    elif event["assigned_event"] is not None:
        print(colored("{} was assigned to an agent...".format(description), "green"))
    elif event["container_started_event"] is not None:
        print(colored("Container of {} has started...".format(description), "green"))
    elif event["service_ready_event"] is not None:
        pass  # Ignore this message.
    elif event["terminate_request_event"] is not None:
        print(colored("{} was requested to terminate...".format(description), "red"))
    elif event["exited_event"] is not None:
        # TODO: Non-success exit statuses should be red
        stat = event["exited_event"]
        print(colored("{} was terminated: {}".format(description, stat), "green"))
        pass
    elif event["log_event"] is not None:
        print(event["log_event"], flush=True)
    else:
        raise ValueError("unexpected event: {}".format(event))
