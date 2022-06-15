import json
import subprocess
import time
from pathlib import Path
from typing import Any, Dict, Iterator, Tuple

import pytest

from determined.common.api.bindings import determinedexperimentv1State as EXP_STATE
from tests import config as conf
from tests import experiment as exp

from .managed_cluster import ManagedCluster
from .utils import get_command_info, run_command, wait_for_command_state

@pytest.mark.managed_devcluster
def test_priortity_scheduler_noop_experiment(managed_cluster: ManagedCluster) -> None:
    managed_cluster.ensure_agent_ok()
    experiment_id1 = exp.run_basic_test(
    conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
)

    trials = exp.experiment_trials(experiment_id1)
    assert len(trials) == 1


@pytest.mark.managed_devcluster
def test_priortity_scheduler_noop_command(managed_cluster: ManagedCluster) -> None:
    managed_cluster.ensure_agent_ok()
    # det cmd run --config resources.slots=1 --config resources.priority=1 'sleep 1
