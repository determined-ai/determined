class Reducer:
    """
    A two-stage hierarchical reduction.  For a simpler interface, see SimpleReducer, below.

    Unlike the current internal reducers, both steps will always run, and users who want a simpler
    interface will let the first step be an identity function.
    """
    # TODO: add a Reducer.reset() method, or just build a new Reducer instance every cycle?

    @abc.abstractmethod
    def accumulate(self, metric):
        """
        Accumulate a metric.  For simple reducers like sums or means, you might just add the metric
        to an accumulator.  For complex reducers, you may have to store the original values in
        memory.
        """
        pass

    @abc.abstractmethod
    def per_slot_reduce(self, metrics):
        """
        d-train optimization: do the bulk of your reduction on each GPU.

        metrics is literally just a list of the metrics your trial returned, no collation at all.

        This is always called, in single- and multi-gpu settings.
        """
        pass

    @abc.abstractmethod
    def cross_slot_reduce(self, per_slot_metrics):
        """
        This is always called, in single- and multi-gpu settings.

        per_slot_metrics will be the result of per_slot_reduce() on every worker, so
        ``len(per_slot_metrics) == slots_per_trial``.
        """
        pass


class SimpleReducer(det.Reducer):
    """Keep SimpleReducer simple.  SimpleReducer(np.mean) would be valid."""

    def __init__(self, reducer_fn):
        self.reducer_fn = reducer_fn
        self.metrics = []

    def accumulate(self, metric):
        self.metrics += metric

    def per_slot_reduce(self, metrics):
        """Just communicate the full metrics to the chief worker."""
        return self.metrics

    def cross_slot_reduce(self, per_slot_metrics):
        """Flatten the list of metrics from each slot so the final reduction is super simple."""
        flat_metrics = [item for sublist in per_slot_metrics for item in sublist]
        return self.reducer_fn(flat_metrics)
