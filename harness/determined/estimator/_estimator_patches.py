import logging

import tensorflow
from packaging import version

from determined import monkey_patch

# TODO(ryan): remove this check after removing support for TensorFlow 1.13.1.
if version.parse(tensorflow.__version__) >= version.parse("1.14.0"):
    import tensorflow.compat.v1 as tf
else:
    import tensorflow as tf

if version.parse(tf.__version__) < version.parse("1.13.0"):
    from tensorflow.python.estimator.training import _NewCheckpointListenerForEvaluate
else:
    from tensorflow_estimator.python.estimator.training import _NewCheckpointListenerForEvaluate

logging.debug("Applying tf.estimator patches.")


@monkey_patch.monkey_patch_decorator(_NewCheckpointListenerForEvaluate, "_evaluate")
def patch_estimator_eval_on_checkpoint(original, *args, **kwargs):  # type: ignore
    # With a single worker and multiple devices,
    # `tf.estimator.train_and_evaluate` attempts to execute `eval_spec` even if
    # `input_fn` or `steps` is None, which causes an error when evaluating the
    # model function. Apply a monkey-patch to skip the internal function that
    # ultimately runs the evaluation.
    logging.info("Skipping %s(*%s, **%s)", original.__name__, args, kwargs)
