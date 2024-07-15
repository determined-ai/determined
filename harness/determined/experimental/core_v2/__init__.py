import determined
from typing import Optional
from determined import core

# Import these directly to make core_v2 a complete package.
from determined.core import DistributedContext, PreemptMode, TensorboardMode
from determined.experimental.core_v2._core_v2 import (
    Config,
    DefaultConfig,
    UnmanagedConfig,
    init_context,
    init,
    close,
    url_reverse_webui_exp_view,
)
from determined.experimental.core_v2._core_context_v2 import _make_v2_context
from determined.experimental.core_v2._unmanaged import (
    _get_or_create_experiment_and_trial,
    _url_reverse_webui_exp_view,
)

# Core V2 singleton variables.
train = None  # type: Optional[core.TrainContext]
distributed = None  # type: Optional[core.DistributedContext]
preempt = None  # type: Optional[core.PreemptContext]
checkpoint = None  # type: Optional[core.CheckpointContext]
searcher = None  # type: Optional[core.SearcherContext]
info = None  # type: Optional[determined.ClusterInfo]
