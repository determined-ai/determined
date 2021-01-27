from collections import namedtuple
from pathlib import Path
from typing import IO, Any, Dict, Iterable, List, Optional, Tuple

from termcolor import colored

from determined_common import api, context, yaml

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

CommandDescription = namedtuple(
    "CommandDescription", ["id", "owner", "description", "state", "exit_status", "resource_pool"]
)


def describe_command(command: Command) -> CommandDescription:
    return CommandDescription(
        command.id,
        command.owner["username"],
        command.config["description"],
        command.state,
        command.exit_status,
        command.config["resources"].get("resource_pool"),
    )


def _set_nested_config(config: Dict[str, Any], key_path: List[str], value: Any) -> Dict[str, Any]:
    current = config
    for key in key_path[:-1]:
        current = current.setdefault(key, {})
    current[key_path[-1]] = value
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
            config = yaml.safe_load(config_file)

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

    for volume_arg in volumes:
        if ":" not in volume_arg:
            raise ValueError(
                "Could not read volume option '{}'\n\n"
                "Expecting:\n{}".format(volume_arg, VOLUME_DESC)
            )

        host_path, container_path = volume_arg.split(":", maxsplit=1)
        bind_mounts = config.setdefault("bind_mounts", [])
        bind_mounts.append({"host_path": host_path, "container_path": container_path})

    # Use the entrypoint command line argument if an entrypoint has not already
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
) -> Any:
    user_files = []  # type: List[Dict[str, Any]]
    if context_path:
        user_files, _ = context.read_context(context_path)

    return api.post(
        master,
        endpoint,
        body={"config": config, "template": template, "user_files": user_files, "data": data},
    ).json()


def render_event_stream(event: Any) -> None:
    description = event["snapshot"]["config"]["description"]
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
