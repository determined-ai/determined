"""
I can't figure this one out.

tf.estimator.Estimator.add_metrics() requires a (metric_tensor, update_op)
tuple.  Or a tf.keras.Metrics object somehow.

We can't put out a metric_tensor unless we synchronize every batch, which seems
awful.  So that's out.

How does it interact with the tf.keras.Metrics object though?

Would it be possible to use tf.compat.v1.py_func somehow?

Do we support py_funcs in general?
"""
