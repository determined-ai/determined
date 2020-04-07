from typing import Any, Dict, List, Optional, cast

import determined as det
from determined import keras


def init(
    config: Optional[Dict[str, Any]] = None,
    mode: det.Mode = det.Mode.SUBMIT,
    context_dir: str = "",
    command: Optional[List[str]] = None,
    master_url: Optional[str] = None,
) -> keras.TFKerasNativeContext:
    return cast(
        keras.TFKerasNativeContext,
        det.init_native(
            controller_cls=keras.TFKerasTrialController,
            native_context_cls=keras.TFKerasNativeContext,
            config=config,
            mode=mode,
            context_dir=context_dir,
            command=command,
            master_url=master_url,
        ),
    )
