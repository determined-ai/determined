import warnings

from determined.cli import *  # noqa
from .__version__ import __version__

warnings.warn(
    "determined_cli package is deprecated, please use determined.cli instead.", FutureWarning
)
