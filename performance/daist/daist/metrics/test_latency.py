import logging
import math
import numpy as np
import random
import time
from typing import List
from unittest import skipIf

from determined.experimental import Experiment

from ..models.metric_latency import Hist, MetricLatencyOpHist, OpHist, SeqSweep
from ..utils import flags
from . import model_def
from .base import BaseMetricsTest
from .model_def import MetricKey

logger = logging.getLogger(__name__)


class Test(BaseMetricsTest):
    def _run_latency_experiment(self,
                                metric_count: int,
                                concurrency: int,
                                samples: int) -> Experiment:

        cfg = self._cfg.format(name=f'{self.id()}',
                               metric_count=metric_count,
                               param1=model_def.Param.CONCURRENCY,
                               param1_minval=1,
                               param1_maxval=concurrency+1,
                               param1_count=concurrency,
                               searcher_max_length=samples,
                               searcher_max_concurrent_trials=concurrency)

        return self._run_experiment(cfg)

    @staticmethod
    def _read_experiment_validation_metrics(experiment:  Experiment, metric_key: MetricKey.type_)\
            -> List[float]:
        metrics = list()

        for trial in experiment.iter_trials():
            # .. todo:: Trials will not have validation metrics if they fail. Note, an experiment
            #           may pass even if there are failing trials within.
            #           See: https://hpe-aiatscale.atlassian.net/browse/SCALE-30
            try:
                metrics.extend(list(trial.iter_metrics('validation'))[0].metrics[metric_key])
            except IndexError as err:
                logger.exception(f'Failed to read validation metrics from trial ID {trial.id}.')
                raise err

        return metrics

    @staticmethod
    def _read_experiment_training_metrics(experiment: Experiment):
        metrics = list()
        for trial in experiment.iter_trials():
            # .. todo:: Trials will not have validation metrics if they fail. Note, an experiment
            #           may pass even if there are failing trials within.
            #           See: https://hpe-aiatscale.atlassian.net/browse/SCALE-30
            try:
                metrics.append(list(trial.iter_metrics('training')))
            except IndexError as err:
                logger.exception(f'Failed to read validation metrics from trial ID {trial.id}.')
                raise err
        return metrics

    def test_concurrency(self):
        debug = False
        concurrency_to_test = (32, 16, 8, 4, 2, 1)
        samples = 128
        op_hist = MetricLatencyOpHist()

        for concurrency in concurrency_to_test:
            if debug:
                write_latencies = np.random.normal(loc=10, scale=.5, size=samples * concurrency)
                read_latencies = np.random.normal(loc=8, scale=.5, size=samples * concurrency)
                write_hist = Hist(samples=write_latencies)
                write_hist.save_plot(MetricKey.WRITE, concurrency, self.id())
                read_hist = Hist(samples=read_latencies)
                read_hist.save_plot(MetricKey.READ, concurrency, self.id())
                op_hist.write[concurrency] = write_hist
                op_hist.read[concurrency] = read_hist
            else:
                experiment = self._run_latency_experiment(
                    MetricLatencyOpHist.TestParamVal.METRIC_COUNT, concurrency, samples)
                write_latencies = self._read_experiment_validation_metrics(experiment,
                                                                           MetricKey.WRITE)

                write_hist = Hist(samples=write_latencies)
                write_hist.save_plot(OpHist.Key.WRITE, concurrency, self.id())
                op_hist.write[concurrency] = write_hist

        op_hist.save_to_results(self.id())
        self.assertFalse(debug)

    def test_sequential(self):
        # A sequence from 1 byte to 256KiB with 11 logarithmically equally spaced metric counts
        metric_counts = self._log_range(1, 256*1024, 11)
        self._test_sequential(metric_counts)

    @skipIf(not flags.SCALE_36, 'Flagged off.')
    def test_scale_36(self):
        """
        The metric counts being
        """
        self._test_sequential([1024*1024, 2*1024*1024, 4*1024*1024])

    @staticmethod
    def _log_range(low: int, high: int, count: int) -> List[int]:
        """
        Credit: https://stackoverflow.com/a/17674783/1144204
        """
        values = list()
        gap = (math.log(high) - math.log(low)) / count
        values.append(low*math.exp(gap))
        for ii in range(count)[1:]:
            values.append(values[ii-1]*math.exp(gap))

        return [int(value) for value in values]

    def _test_sequential(self, metric_counts: List[int]):
        """
        .. todo:: Hit the following error when attempting 2097152 metric floats (at 8 bytes per
            float, that comes to 16MiB of user metrics)::

                Error reporting metrics: failed to exec transaction (add trial metrics training):
                updating trial total batches: ERROR: total size of jsonb object elements exceeds the
                maximum of 268435455 bytes (SQLSTATE 54000)

            also hit the following with 1048576 metric floats (at 8 bytes per float, that comes
            to 8MiB of user metrics.)::

                determined.common.api.errors.APIException: grpc: received message larger than max
                (154078479 vs. 134217728)

            Also, the server was in a hang state.
        """
        debug = False
        read_latencies = list()
        write_latencies = list()
        if debug:
            write_latencies = [random.random() for _ in metric_counts]
            read_latencies = [random.random() for _ in metric_counts]
        else:
            for metric_count in metric_counts:
                experiment = self._run_latency_experiment(metric_count, concurrency=1, samples=1)
                write_latencies.extend(
                    self._read_experiment_validation_metrics(experiment, MetricKey.WRITE))
                start_seconds = time.time()
                _ = self._read_experiment_training_metrics(experiment)
                read_latencies.append(time.time() - start_seconds)

        seq_sweep = SeqSweep(metric_counts=metric_counts,
                             write=write_latencies, read=read_latencies)
        seq_sweep.save_to_results(self.id())
        self.assertFalse(debug)
