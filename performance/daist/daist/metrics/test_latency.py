from typing import List
import logging

from . import model_def
from .base import BaseMetricsTest
from .model_def import MetricKey
from ..models.metric_latency import MetricLatencyOpHist, Hist

logger = logging.getLogger(__name__)


class Test(BaseMetricsTest):
    # _CONCURRENCY_TO_TEST = (1,)
    _CONCURRENCY_TO_TEST = (32, 16, 8, 4, 2, 1)
    # _CONCURRENCY_TO_TEST = (512, 256, 128, 64, 32, 16, 8, 4, 2, 1)

    _SAMPLES = 1024

    def _run_latency_experiment(self, concurrency: int, samples: int) -> List[float]:

        cfg = self._cfg.format(name=f'{self.id()}',
                               metric_count=MetricLatencyOpHist.TestParamVal.METRIC_COUNT,
                               param1=model_def.Param.CONCURRENCY,
                               param1_minval=1,
                               param1_maxval=concurrency+1,
                               param1_count=concurrency,
                               searcher_max_length=samples,
                               searcher_max_concurrent_trials=concurrency)

        experiment = self._run_experiment(cfg)

        write_latencies = list()

        for trial_idx, trial in enumerate(experiment.iter_trials()):
            # .. todo:: Trials will not have validation metrics if they fail. Note, an experiment
            #           may pass even if there are failing trials within.
            #           See: https://hpe-aiatscale.atlassian.net/browse/SCALE-30
            try:
                metrics = list(trial.iter_metrics('validation'))[0].metrics
            except IndexError as err:
                logger.exception(f'Failed to read validation metrics from trial ID {trial.id}.')
                raise err
            write_latencies.extend(metrics[MetricKey.WRITE])

        return write_latencies

    def test(self):
        op_hist = MetricLatencyOpHist()
        for concurrency in self._CONCURRENCY_TO_TEST:
            write_latencies = self._run_latency_experiment(concurrency, self._SAMPLES)

            # write_latencies = np.random.normal(loc=10, scale=.5, size=1024*concurrency)
            # read_latencies = np.random.normal(loc=8, scale=.5, size=1024*concurrency)

            op_hist.write[concurrency] = Hist(samples=write_latencies)
            # op_hist.read[concurrency] = Hist(samples=read_latencies)
        # .. todo:: save the plots as they become available to capture partial data on test error.
        op_hist.make_plots(self.id())
        op_hist.show()
        op_hist.save_to_results(self.id())
