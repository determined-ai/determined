try:
    from ruamel import yaml
except ModuleNotFoundError:
    # Inexplicably, sometimes ruamel.yaml is pacakged as ruamel_yaml instead.
    import ruamel_yaml as yaml  # type: ignore

from . import api, check, constants, context, requests, storage, types, util
from ._logging import set_logger
from .__version__ import __version__
