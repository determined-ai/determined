from contextlib import redirect_stdout, redirect_stderr
from pathlib import Path
from tempfile import NamedTemporaryFile
from typing import cast, IO
from unittest import TestCase
import logging
import time

from determined.common.api import bindings
from determined.common.api.errors import APIException
from determined.experimental import client, Experiment


from ..models.session import session
from ..utils.stream_to_logger import StreamToLogger

API_EXCEPTION_RETRIES = 10
API_EXCEPTION_RETRY_DELAY_SEC = 1
PATH_TO_HERE = Path(__file__).parent.resolve()
logger = logging.getLogger(__name__)


class BaseMetricsTest(TestCase):
    _cfg = """\
name: {name}
entrypoint: python3 model_def.py launcher

environment:
  image: determinedai/environments:py-3.9-pytorch-1.12-tf-2.11-cpu-0.27.1
  environment_variables:
    # This is a workaround for a dead lock issue involving multiple layers of NFS mounts.
    - DET_DEBUG_CONFIG_PATH=/tmp

hyperparameters:
  checkpoint_size: 4096
  metric_count: {metric_count}
  {param1}:
    type: int
    minval: {param1_minval}
    maxval: {param1_maxval}
    count: {param1_count}

optimizations:
  average_training_metrics: False

resources:
  slots_per_trial: 1
  
  # This may or may not be needed
  # resource_pool: compute_pool  

searcher:
  name: grid
  metric: "{param1}"
  max_length: {searcher_max_length}
  max_concurrent_trials: {searcher_max_concurrent_trials}

max_restarts: 0"""

    @classmethod
    def setUpClass(cls):
        client.login(session.determined.host,
                     session.determined.user,
                     session.determined.password)

    @classmethod
    def tearDownClass(cls):
        client.logout()

    @staticmethod
    def create_experiment(cfg: str) -> Experiment:
        with NamedTemporaryFile('w+') as tmp_file:
            path_to_tmp_file = Path(tmp_file.name)
            path_to_tmp_file.write_text(cfg)
            tmp_file.flush()
            with redirect_stdout(cast(IO[str], StreamToLogger(logger.debug))):
                return client.create_experiment(config=path_to_tmp_file,
                                                model_dir=str(PATH_TO_HERE))

    def _run_experiment(self, cfg: str) -> Experiment:
        experiment = self.create_experiment(cfg)
        logger.info(f'Running experiment {experiment.id}')
        with redirect_stdout(cast(IO[str], StreamToLogger(logger.info))), \
             redirect_stderr(cast(IO[str], StreamToLogger(logger.info))):
            # .. todo:: Add a timeout_s parameter.

            for retry in range(API_EXCEPTION_RETRIES):
                # The following was observed in this retry loop:
                # - https://hpe-aiatscale.atlassian.net/browse/MD-452

                try:
                    experiment.wait(interval=0.25)
                    break
                except APIException:
                    logger.exception(f'Retry waiting on experiment {experiment.id}.')
                    time.sleep(API_EXCEPTION_RETRY_DELAY_SEC)
                    continue
            else:
                raise Exception(f'Exhausted {API_EXCEPTION_RETRIES} retries waiting for '
                                f'experiment {experiment.id}')

        if client.ExperimentState.COMPLETED != experiment.state:
            raise Exception(f'Expected the final experiment ID {experiment.id} state to be '
                            f'{bindings.experimentv1State.COMPLETED}, observed {experiment.state}')
        return experiment
