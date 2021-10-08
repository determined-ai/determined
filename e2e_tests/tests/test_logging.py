import contextlib
import json
import operator
import os
import subprocess
import tempfile
import threading
import time
from typing import Any, Dict, Iterator, List, Optional, Set, cast

import numpy as np
import pytest
import yaml

from determined import errors
from determined.common import storage
from determined.experimental import Determined, ModelSortBy
from tests import config as conf
from tests import experiment as exp
from tests.fixtures.metric_maker.metric_maker import structure_equal, structure_to_metrics


@pytest.mark.e2e_cpu  # type: ignore
def test_trial_logs() -> None:
    experiment_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    trial_id = exp.experiment_trials(experiment_id)[0]["id"]
    subprocess.check_call(["det", "-m", conf.make_master_url(), "trial", "logs", str(trial_id)])
    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "trial", "logs", "--head", "10", str(trial_id)],
    )
    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "trial", "logs", "--tail", "10", str(trial_id)],
    )

@pytest.mark.e2e_cpu # type: ignore
def test_task_logs() -> None:
    
