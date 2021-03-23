import warnings

from determined.deploy import *  # noqa
from .__version__ import __version__

warnings.warn(
    "determined_deploy package is deprecated, please use determined.deploy instead.", FutureWarning
)
