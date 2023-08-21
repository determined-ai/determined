import warnings

# TODO: delete all of these when det.experimental.client is removed.
from determined.common.api import Session
from determined.common.experimental.checkpoint import Checkpoint
from determined.common.experimental.determined import Determined
from determined.common.experimental.experiment import ExperimentReference
from determined.common.experimental.trial import TrialReference, TrialSortBy, TrialOrderBy
from determined.common.experimental.model import Model, ModelOrderBy, ModelSortBy, ModelVersion
from determined.common.experimental.metrics import (
    TrialMetrics,
    TrainingMetrics,
    ValidationMetrics,
)


warnings.warn(
    "The 'determined.common.experimental' module is deprecated and will be removed "
    "in future versions. Please import from 'determined.experimental' instead. "
    "Example: `from determined.experimental import Determined`",
    FutureWarning,
    stacklevel=2,
)
