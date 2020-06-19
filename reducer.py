"""
## Description

We've been talking about custom reducers pretty seriously since mid-April.  Our
major customers have asked for it and we should do it.  This is a proposal for
how we could support them.

I am hoping to have a discussion about this API at the next ml-ag meeting.

**Conversation regarding validation metrics (#ml-ag, April 16):**
    https://determined-ai.slack.com/archives/CSLAGUF3M/p1587050305213200

**Conversation regarding training metrics (#ml-ag, April 17):**
    https://determined-ai.slack.com/archives/CSLAGUF3M/p1587138289237300


The API below addresses validation metrics, not training metrics, but it
applies the following ideas proposed by @brain-good, @armandmcqueen , and
@aaron276h during the discussion of training metrics:

from @brain-good:

>[we should] combine all the metrics and use a single function to aggregate the
metrics.  It’s a lot easier to reason as a user.

(see det.SimpleReducer, designed to meet this need)

from @armandmcqueen :

> Can we come up with a more user-friendly approach than concatenation - dict
of lists/2d array?

(see the det.Reducer, which doesn't do any magic and gives access full access
to raw metrics)

from @aaron276h:

> If we keep [@brain-good's strategy vs a hierarchichal reduce] configurable,
even if there is a noticable performance hits shouldn't be a big a problem

(det.SimpleReducer as a subclass of det.Reducer is designed to meet this need)


(I was supposed to do some profiling work to measure the performance overhead
of @brain-good's strategy vs a hierarchichal reduce strategy, but the truth is
that I have not yet had time, and the more I thought about it the more I like
@aaron276h's idea of making it configurable)
"""


class Reducer:
    """
    A two-stage hierarchical reduction.  For a simpler interface, see SimpleReducer, below.

    Unlike the current internal reducers, both steps will always run, and users who want a simpler
    interface will let the first step be an identity function.
    """

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


class MyAverageReducer(det.Reducer):
    """Example implementation of a Reducer"""

    def per_slot_reduce(self, metrics):
        # Assume metrics is just a list of scalars
        return [sum(metrics) / len(metrics), len(metrics)]

    def cross_slot_reduce(self, per_slot_metrics):
        """
        Note that this step has to be a weighted average, even though the overall average is not a
        weighted average.  This is the weird part of the hierarchical reduce.
        """
        weight_sum = sum(m[1] for m in per_slot_metrics)
        return sum(val * weight for val, weight in per_slot_metrics) / weight_sum


class MyWeightedAverageReducer(det.Reducer):
    """Example implementation of a Reducer"""

    def per_slot_reduce(self, metrics):
        # Assume metrics is a list of [val, weight] for each batch
        weight_sum = sum(m[1] for m in metrics)
        return [sum(val * weight for val, weight in metrics), weight_sum]

    def cross_slot_reduce(self, per_slot_metrics):
        weight_sum = sum(m[1] for m in per_slot_metrics)
        return sum(v * w for v, w in per_slot_metrics) / weight_sum

#

class SimpleReducer(det.Reducer):
    """Offer a simplest-possible alternative that does the full reduction in a single step."""

    def per_slot_reduce(self, metrics):
        """Just communicate the full metrics to the chief worker."""
        return metrics

    def cross_slot_reduce(self, per_slot_metrics):
        """Flatten the list of metrics from each slot so the final reduction is super simple."""
        flat_metrics = [item for sublist in per_slot_metrics for item in sublist]
        return self.reduce(flat_metrics)

    @abc.abstractmethod
    def reduce(metrics):
        """User provides this."""
        pass


class MyAverageSimpleReducer(det.SimpleReducer):
    """This is about as simple as a user-defined custom reducer can be."""
    def reduce(metrics):
        return sum(metrics) / len(metrics)


# How would we integrate with EstimatorTrial?
# We would integrate via custom SessionRunHooks. Something like this:

class EstimatorTrialContext:
    ...
    def allreduce_metrics(self, reducer, metrics):
        """
        Why expose this as part of the context?  We have customers asking for the ability
        to see metrics in tensorboard which are being calculated by hand in EvalHooks, but which
        are not being shared across GPUs, so their tensorboard metrics are not quite right.

        Arguably, in those cases we don't need a two-step reducer like det.Reducer would be;
        they would be able to do the per_slot_reduce() on their own in their EvalHook.

        Frankly, I'm not convinced that this should be exposed to users; it seems like they could
        skip implementing their custom EvalHook with the reducer interface.
        """

        this_slot_metrics = reducer.per_slot_reduce(metrics)

        # Handle the single-slot case.
        if self.distributed.size() == 0:
            return reducer.cross_slot_reduce(this_slot_metrics)

        # Handle the chief case.
        if self.distributed.rank() == 0:
            per_slot_metrics = chief_gather_per_slot_metrics(this_slot_metrics)
            reduced = self.reducer.cross_slot_reduce(per_slot_metrics)
            chief_distribute_reduce_metrics(reduced)
        else:
            reduced = worker_allreduce_metrics(this_slot_metrics)
        return reduced


class ReducerEvalHook(tf.compat.v1.train.SessionRunHook):
    """
    Apply the reducer to the outputs of the fetches from each batch, and write the final result to
    metric_name in tensorboard.

    Users would include an instance of ReducerEvalHook as part of their EvalSpec.
    """

    def __init__(self, context, fetches, reducer, metric_name):
        self.context = context
        self.fetches = fetches
        self.reducer = reducer
        self.metric_name = metric_name

    def begin(self):
        self.batch_metrics = []

    def before_run(self, run_contex):
        return tf.estimator.SessionRunArgs(fetches=self.fetches)

    def after_run(self, run_contex, run_values):
        self.per_slot_metrics.append(run_values.results)

    def end(self, session):
        reduced = self.context.allreduce_metrics(self.reducer, self.batch_metrics)

        write_metric_to_tensorboard(self.metric_name, reducer)


class MyLinearEstimator(estimator.EstimatorTrial):
    ...

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        def fn():
            ...

        global_step_metric_hook = det.estimator.ReducerEvalHook(
            context=self.context,
            fetches="my_variable:0",
            reducer=MyReducer(),
            metric_name="my_metric",
        )

        return tf.estimator.EvalSpec(fn, hooks=[global_step_metric_hook])

        # Shiyuan says: maybe make this a separate callback and hide the ReducerEvalHook thing?



# How would we integrate with PyTorchTrial?
#
# Basically exactly how we currently support selecting reducers, only we would accept instances
# of Reducer instead of the det.pytorch.Reducer enum (we'd could easily be backwards
# compatible though):

class MyPytorchTrial(PyTorchTrial):
    ...

    def evaluation_reducer(self):
        return {"error": MyErrorReducer(), "accuracy": MyAccuracyReducer()}




#######################
# Alternate idea: it seems like the Reducer interface described above is simple, but it enforces
# poor memory usage patterns.  What if we added an accumulate() method?  Things like confusion
# matrices could be accumualated in very memory-efficient ways.

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


class MyAverageReducer(det.Reducer):
    """Example implementation of a Reducer"""
    def __init__(self):
        self.sum = None
        self.count = 0

    def accumulate(self, metric):
        if self.sum is None:
            self.sum =f metric
        else:
            self.sum += metric
        self.count += 1

    def per_slot_reduce(self, metrics):
        return self.sum, self.count

    def cross_slot_reduce(self, per_slot_metrics):
        total_count = sum(m[1] for m in per_slot_metrics)
        return sum(val * count for val, count in per_slot_metrics) / total_count


class SimpleReducer(det.Reducer):
    """Keep SimpleReducer simple."""
    def __init__(self):
        self.metrics = []

    def accumulate(self, metric):
        self.metrics += metric

    def per_slot_reduce(self, metrics):
        """Just communicate the full metrics to the chief worker."""
        return self.metrics

    def cross_slot_reduce(self, per_slot_metrics):
        """Flatten the list of metrics from each slot so the final reduction is super simple."""
        flat_metrics = [item for sublist in per_slot_metrics for item in sublist]
        return self.reduce(flat_metrics)

    @abc.abstractmethod
    def reduce(metrics):
        """User provides this."""
        pass
