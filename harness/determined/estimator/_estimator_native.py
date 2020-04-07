from typing import Any, Dict, List, Optional, cast

import determined as det
from determined import estimator


def init(
    config: Optional[Dict[str, Any]] = None,
    mode: det.Mode = det.Mode.SUBMIT,
    context_dir: str = "",
    command: Optional[List[str]] = None,
    master_url: Optional[str] = None,
) -> estimator.EstimatorNativeContext:
    return cast(
        estimator.EstimatorNativeContext,
        det.init_native(
            controller_cls=estimator.EstimatorTrialController,
            native_context_cls=estimator.EstimatorNativeContext,
            config=config,
            mode=mode,
            context_dir=context_dir,
            command=command,
            master_url=master_url,
        ),
    )
