from typing import Optional

import termcolor

import determined as det


class PreflightFailure(Exception):
    pass


class MasterTimeoutExpired(Exception):
    pass


def warn_version_mistmatch(requestd_version: Optional[str]) -> None:
    """
    check for and warn about version compatibility between the `det` CLI and
    the requested det version.
    """
    if not requestd_version:
        return
    print(
        termcolor.colored(
            f"Warning: The specified --det-version ({requestd_version}) does not match the "
            f"the current `det` CLI version ({det.__version__}), proceed with caution.",
            "You should use a matching version of det CLI. ",
            "https://docs.determined.ai/latest/tools/cli/cli-ug.html#determined-cli",
            "yellow",
        )
    )
