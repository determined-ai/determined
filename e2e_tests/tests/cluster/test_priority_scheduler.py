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
from .utils import get_command_info, run_command, run_command_set_priority, wait_for_command_state

@pytest.mark.managed_devcluster
def test_priortity_scheduler_noop_experiment(managed_cluster_priority_scheduler: ManagedCluster) -> None:
    managed_cluster_priority_scheduler.ensure_agent_ok()
    # uses the default priority set in cluster config 
    experiment_id1 = exp.run_basic_test(
    conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1)
    # uses explicit priority 
    experiment_id2 = exp.run_basic_test(
    conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1, priority=50)

@pytest.mark.managed_devcluster
def test_priortity_scheduler_noop_command(managed_cluster_priority_scheduler: ManagedCluster) -> None:
    managed_cluster_priority_scheduler.ensure_agent_ok()
    # with slots (and default priority)
    assert run_command(slots=2) == "0"
    # without slots (and default priority)
    assert run_command(slots=0) == "0"
    # explicity priority
    assert run_command_set_priority(slots=0) == "0"


@pytest.mark.managed_devcluster
def test_slots_list_command(managed_cluster_priority_scheduler: ManagedCluster) -> None: 
     managed_cluster_priority_scheduler.ensure_agent_ok()
