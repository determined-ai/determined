from typing import Any, Dict, List, Optional, cast

from determined import estimator, experimental


def init(
    config: Optional[Dict[str, Any]] = None,
    local: bool = False,
    test: bool = False,
    context_dir: str = "",
    command: Optional[List[str]] = None,
    master_url: Optional[str] = None,
) -> estimator.EstimatorNativeContext:
    # TODO: Add a reference to Native tutorial / topic-guide.
    """
    Create a tf.estimator experiment using the Native API.

    Arguments:
        config:
            A dictionary representing the experiment configuration to be
            associated with the experiment.

        local:
            A boolean indicating if training will happen locally. When
            ``False``, the experiment will be submitted to the Determined
            cluster. Defaults to ``False``.

        test:
            A boolean indicating if the experiment should be shortened to a
            minimal loop of training, validation, and checkpointing.
            ``test=True`` is useful quick iterating during model porting or
            debugging because common errors will surface more quickly.
            Defaults to ``False``.

        context_dir:
            A string filepath that defines the context directory. All model
            code will be executed with this as the current working directory.

            When ``local=False``, this argument is required. All files in this
            directory will be uploaded to the Determined cluster. The total
            size of this directory must be under 96 MB.

            When ``local=True``, this argument is optional and assumed to be
            the current working directory by default.

        command:
            A list of strings that is used as the entrypoint of the training
            script in the Determined task environment. When executing this
            function via a python script, this argument is inferred to be
            ``sys.argv`` by default. When executing this function via IPython
            or Jupyter notebook, this argument is required.

        master_url:
            An optional string to use as the Determined master URL when
            ``local=False``. If not specified, will be inferred from the
            environment variable ``DET_MASTER``.

    Returns:
        :py:class:`determined.estimator.EstimatorNativeContext`
    """

    if local and not test:
        raise NotImplementedError(
            "estimator.init(local=True, test=False) is not yet implemented. Please set local=False "
            "or test=True."
        )

    return cast(
        estimator.EstimatorNativeContext,
        experimental.init_native(
            controller_cls=estimator.EstimatorTrialController,
            native_context_cls=estimator.EstimatorNativeContext,
            config=config,
            local=local,
            test=test,
            context_dir=context_dir,
            command=command,
            master_url=master_url,
        ),
    )
