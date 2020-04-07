"""
The issue: if a session is created with a ConfigProto specifying inter_op_parallelism_threads=1,
any tf.data.Iterator which was created from a tf.data.TFRecordDataset cannot be restored; it just
hangs forever.

See TensorFlow bug: https://github.com/tensorflow/tensorflow/issues/29937

The workaround: it is possible to create a session capable of using different thread pools on
different invocations of sess.run(). We can modify any ConfigProto to guarantee that we always have
a larger-than-one-sized thread pool available to us at any time.

To accomplish this, we check that the ConfigProto given by the user either explicitly specifies an
appropriate thread pool, or we alter the ConfigProto to include one, which we will explicitly
select during iterator restoration.

In olden times, inter_op_parallelism_threads specified the size of the thread pool for the session.
Now, session_inter_op_thread_pool is used to specify a list of thread pools, which can be selected
at session.run()-time using a tf.RunOptions object with the inter_op_thread_pool option, which
specifies the index of the session_inter_op_thread_pool list to use.

The session_inter_op_thread_pool mechanism is basically undocumented.  See the commit where it was
added:

    https://github.com/tensorflow/tensorflow/commit/49ec6ff0017a436f5

The simplest and most robust solution here is to always upgrade a user's ConfigProto to use the
newer session_inter_op_thread_pool mechanism, add a pool of appropriate size if necessary, and
explicitly request that pool during Saver.restore().

However, tf.train.Saver.restore() does not expose a way to specify a tf.RunOptions object for its
call to sess.run(), so we patch restore() to accept/pass a tf.RunOptions object.

This patch is not strictly required; we could instead modify the default thread pool for the
session to always be multi-threaded, and then in every call to sess.run() we could specify a
non-default thread pool to use, but that would be confusing if we ever handed the user back the
tf_session from TensorFlowTrial, due to the fact that we would have silently modified its default
behavior to handle this fairly obscure bug.
"""

import tensorflow
from packaging import version
from tensorflow.python.eager import context
from tensorflow.python.framework import errors
from tensorflow.python.platform import tf_logging as logging
from tensorflow.python.training import checkpoint_management
from tensorflow.python.training.saver import (
    _wrap_restore_error_with_msg,
    object_graph_key_mapping,
    saver_from_object_based_checkpoint,
)
from tensorflow.python.util import compat

# Handle TensorFlow compatibility issues.
if version.parse(tensorflow.__version__) >= version.parse("1.14.0"):
    import tensorflow.compat.v1 as tf
else:
    import tensorflow as tf


# taken from tensorflow/tensorflow/python/training/saver.py:1243
def patched_restore(self, sess, save_path, options=None):  # type: ignore
    """
    Restores previously saved variables.

    This method runs the ops added by the constructor for restoring variables.
    It requires a session in which the graph was launched.  The variables to
    restore do not have to have been initialized, as restoring is itself a way
    to initialize variables.

    The `save_path` argument is typically a value previously returned from a
    `save()` call, or a call to `latest_checkpoint()`.

    Args:
      sess: A `Session` to use to restore the parameters. None in eager mode.
      save_path: Path where parameters were previously saved.

    Raises:
      ValueError: If save_path is None or not a valid checkpoint.
    """
    if self._is_empty:
        return
    if save_path is None:
        raise ValueError("Can't load save_path when it is None.")

    checkpoint_prefix = compat.as_text(save_path)
    if not checkpoint_management.checkpoint_exists(checkpoint_prefix):
        raise ValueError("The passed save_path is not a valid checkpoint: " + checkpoint_prefix)

    logging.info("Restoring parameters from %s", checkpoint_prefix)
    try:
        if context.executing_eagerly():
            self._build_eager(save_path, build_save=False, build_restore=True)
        else:
            sess.run(
                self.saver_def.restore_op_name,
                {self.saver_def.filename_tensor_name: save_path},
                options=options,
            )
    except errors.NotFoundError as err:
        # There are three common conditions that might cause this error:
        # 0. The file is missing. We ignore here, as this is checked above.
        # 1. This is an object-based checkpoint trying name-based loading.
        # 2. The graph has been altered and a variable or other name is missing.

        # 1. The checkpoint would not be loaded successfully as is. Try to parse
        # it as an object-based checkpoint.
        try:
            names_to_keys = object_graph_key_mapping(save_path)
        except errors.NotFoundError:
            # 2. This is not an object-based checkpoint, which likely means there
            # is a graph mismatch. Re-raise the original error with
            # a helpful message (b/110263146)
            raise _wrap_restore_error_with_msg(
                err, "a Variable name or other graph key that is missing"
            )

        # This is an object-based checkpoint. We'll print a warning and then do
        # the restore.
        logging.warning(
            "Restoring an object-based checkpoint using a name-based saver. This "
            "may be somewhat fragile, and will re-build the Saver. Instead, "
            "consider loading object-based checkpoints using "
            "tf.train.Checkpoint()."
        )
        self._object_restore_saver = saver_from_object_based_checkpoint(
            checkpoint_path=save_path,
            var_list=self._var_list,
            builder=self._builder,
            names_to_keys=names_to_keys,
            cached_saver=self._object_restore_saver,
        )
        self._object_restore_saver.restore(sess=sess, save_path=save_path, options=options)
    except errors.InvalidArgumentError as err:
        # There is a mismatch between the graph and the checkpoint being loaded.
        # We add a more reasonable error message here to help users (b/110263146)
        raise _wrap_restore_error_with_msg(
            err, "a mismatch between the current graph and the graph"
        )


# Fortunately, tf.train.Saver.restore is identical in all versions of TensorFlow we currently
# support (the only change looks like a change in linter preferences). If you going to change the
# allowed versions here, you should manually check the code diffs for this file.
if version.parse(tf.__version__) < version.parse("1.12.0") or version.parse(
    tf.__version__
) > version.parse("1.15.0"):
    pass
else:
    tf.train.Saver.restore = patched_restore
