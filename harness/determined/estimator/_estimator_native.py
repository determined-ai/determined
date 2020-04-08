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
    """
    Create a tf.estimator experiment using the Native API.

    .. TODO: Add a reference to Native tutorial / topic-guide.

    Arguments:
        config:
            A dictionary representing the experiment configuration to be
            associated with the experiment.
        mode:
            The :py:class:`determined.Mode` used when creating an experiment

            1. ``Mode.SUBMIT`` (default): Submit the experiment to a remote
            Determined cluster.

            2. ``Mode.TEST`` (default): Test the experiment in the calling
            Python process for development / debugging purposes. Run through a
            minimal loop of training, validation, and checkpointing steps.

        context_dir:
            A string filepath that defines the context directory. In submit
            mode, all files in this directory will be uploaded to the
            Determined cluster.
        command:
            A list of strings that is used as the entrypoint of the training
            script in the Determined task environment. When executing this
            function via a python script, this argument is inferred to be
            ``sys.argv`` by default. When executing this function via IPython
            or Jupyter notebook, this argument is required.
        master_url:
            An optional string to use as the Determined master URL in submit
            mode. Will default to the value of environment variable
            ``DET_MASTER`` if not provided.

    Returns:
        :py:class:`determined.estimator.EstimatorNativeContext`
    """
    return cast(
        estimator.EstimatorNativeContext,
        det._init_native(
            controller_cls=estimator.EstimatorTrialController,
            native_context_cls=estimator.EstimatorNativeContext,
            config=config,
            mode=mode,
            context_dir=context_dir,
            command=command,
            master_url=master_url,
        ),
    )
