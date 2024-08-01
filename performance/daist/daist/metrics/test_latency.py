import logging
import time
from typing import List, Tuple

from determined.experimental import Experiment

from ..models.metric_latency import Hist, MetricLatencyOpHist
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
    def _read_experiment_validation_metrics(experiment: Experiment, metric_key: MetricKey.type_)\
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
        concurrency_to_test = (32, 16, 8, 4, 2, 1)
        samples = 1024
        op_hist = MetricLatencyOpHist()
        for concurrency in concurrency_to_test:
            experiment = self._run_latency_experiment(
                MetricLatencyOpHist.TestParamVal.METRIC_COUNT, concurrency, samples)
            write_latencies = self._read_experiment_validation_metrics(experiment, MetricKey.WRITE)

            # write_latencies = np.random.normal(loc=10, scale=.5, size=1024*concurrency)
            # read_latencies = np.random.normal(loc=8, scale=.5, size=1024*concurrency)

            op_hist.write[concurrency] = Hist(samples=write_latencies)
            # op_hist.read[concurrency] = Hist(samples=read_latencies)
        # .. todo:: save the plots as they become available to capture partial data on test error.
        op_hist.make_plots(self.id())
        op_hist.show()
        op_hist.save_to_results(self.id())

    def test_sequential(self):
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
        # A sweep of [1 byte,  1MiB] where each step multiplies the previous step by 4.
        metric_counts_to_test = (1, 4, 16, 64, 256, 1024, 4096, 65536, 262144)
        # metric_counts_to_test = (1, 8, 64)
        read_latencies = list()
        write_latencies = list()
        for metric_count in metric_counts_to_test:
            experiment = self._run_latency_experiment(metric_count, concurrency=1, samples=1)
            write_latencies.extend(
                self._read_experiment_validation_metrics(experiment, MetricKey.WRITE))
            start_seconds = time.time()
            training_metrics = self._read_experiment_training_metrics(experiment)
            read_latencies.append(time.time() - start_seconds)

        # write_latencies = [0.0011074542999267578, 0.0012280941009521484, 0.0026731491088867188]
        # read_latencies = [0.010850667953491211, 0.014742612838745117, 0.01394796371459961]

        import matplotlib.pyplot as plt
        from matplotlib.axes import Axes

        fig, axs = plt.subplots(2, constrained_layout=True, sharex=True)
        axs: List[Axes]
        ax: Axes = axs[0]
        ax.plot(metric_counts_to_test, write_latencies, marker='.')
        ax.set_xticks(metric_counts_to_test)
        ax.set_title('Write Latency by Metric Count')
        ax.set_ylabel('Time (seconds)')

        ax: Axes = axs[1]
        ax.plot(metric_counts_to_test, read_latencies, marker='.')
        ax.set_xticks(metric_counts_to_test)
        ax.set_title('Read Latency by Metric Count')
        ax.set_xlabel('Metric Count (floats)')

        plt.show()
