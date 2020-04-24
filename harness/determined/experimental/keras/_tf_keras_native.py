from typing import Any, Dict, List, Optional, cast

from determined import experimental, keras


def init(
    config: Optional[Dict[str, Any]] = None,
    mode: experimental.Mode = experimental.Mode.CLUSTER,
    context_dir: str = "",
    command: Optional[List[str]] = None,
    master_url: Optional[str] = None,
) -> keras.TFKerasNativeContext:
    """
    Create a tf.keras experiment using the Native API.

    .. TODO: Add a reference to Native tutorial / topic-guide.

    Arguments:
        config:
            A dictionary representing the experiment configuration to be
            associated with the experiment.
        mode:
            The :py:class:`determined.experimental.Mode` used when creating an
            experiment

            1. ``Mode.CLUSTER`` (default): Submit the experiment to a remote
            Determined cluster.

            2. ``Mode.LOCAL``: Test the experiment in the calling
            Python process for development / debugging purposes. Run through a
            minimal loop of training, validation, and checkpointing steps.

        context_dir:
            A string filepath that defines the context directory. All model
            code will be executed with this as the current working directory.

            In CLUSTER mode, this argument is required. All files in this
            directory will be uploaded to the Determined cluster. The total
            size of this directory must be under 96 MB.

            In LOCAL mode, this argument is optional and assumed to be the
            current working directory by default.
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
        :py:class:`determined.keras.TFKerasNativeContext`
    """
    return cast(
        keras.TFKerasNativeContext,
        experimental._native._init_native(
            controller_cls=keras.TFKerasTrialController,
            native_context_cls=keras.TFKerasNativeContext,
            config=config,
            mode=mode,
            context_dir=context_dir,
            command=command,
            master_url=master_url,
        ),
    )
