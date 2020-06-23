class Metric(tf.keras.Metric):
    # Make it easy? Only support eager execution?

    def __init__(self, base_metric):
        self.base_metric = base_metric
        self.updates = []

    def reset_states(self):
        self.base_metric.reset_states()
        self.updates = []

    def update_state(self, *args, **kwargs):
        self.updates.append((args, kwargs))

    def result(self):
        # Handle the single-slot case.
        if context.distributed.size() == 0:
            for args, kwargs in self.updates:
                self.base_metric.update(*args, **kwargs)
            return self.base_metric.result()

        # Handle the chief case.
        if context.distributed.rank() == 0:
            all_updates = context._chief_gather_updates(self.updates)
            for args, kwargs in all_updates:
                self.base_metric.update(*args, **kwargs)
            result = self.base_metric.result()
            context._chief_distribute_result(result)
        else:
            result = worker_allreduce_metric(self.base_metric)
        return result

"""
risk: can we support this API at all:
    https://www.tensorflow.org/api_docs/python/tf/keras/metrics/Mean

    model = tf.keras.Model(inputs, outputs)
    model.add_metric(tf.keras.metrics.Mean(name='mean_1')(outputs))
    model.compile('sgd', loss='mse')
"""
