from determined.cli._util import (
    output_format_args,
    make_pagination_args,
    default_pagination_args,
    unauth_session,
    session,
    setup_session,
    require_feature_flag,
    print_launch_warnings,
    wait_ntsc_ready,
    warn,
)
from determined.cli._declarative_argparse import (
    Arg,
    ArgsDescription,
    ArgGroup,
    BoolOptArg,
    Cmd,
    Group,
    add_args,
    deprecation_warning,
    help_func,
    generate_aliases,
    make_prefixes,
    string_to_bool,
    wrap_func,
)
from determined.cli.errors import CliError

from determined.common.api import certs as _certs
from typing import Optional as _Optional

# cert is a singleton that we configure very early in the cli's main() function, before any cli
# subcommand handlers are invoked.
cert: _Optional[_certs.Cert] = None
