import contextlib
import io
import os
import sys
from typing import Callable, Dict, Iterator, List, Optional

import determined as det
from tests.experiment import utils


def make_mock_cluster_info(
    container_addrs: List[str],
    container_rank: int,
    num_slots: int,
    latest_checkpoint: Optional[str] = None,
) -> det.ClusterInfo:
    config = utils.make_default_exp_config({}, 100, "loss", None)
    trial_info_mock = det.TrialInfo(
        trial_id=1,
        experiment_id=1,
        trial_seed=0,
        hparams={},
        config=config,
        steps_completed=0,
        trial_run_id=0,
        debug=False,
        inter_node_network_interface=None,
    )
    rendezvous_info_mock = det.RendezvousInfo(
        container_addrs=container_addrs,
        container_rank=container_rank,
        container_slot_counts=[int(num_slots / len(container_addrs)) for _ in container_addrs],
    )
    cluster_info_mock = det.ClusterInfo(
        master_url="localhost",
        cluster_id="clusterId",
        agent_id="agentId",
        slot_ids=list(range(num_slots)),
        task_id="taskId",
        allocation_id="allocationId",
        session_token="sessionToken",
        task_type="TRIAL",
        rendezvous_info=rendezvous_info_mock,
        trial_info=trial_info_mock,
        latest_checkpoint=latest_checkpoint,
    )
    return cluster_info_mock


@contextlib.contextmanager
def set_mock_cluster_info(
    container_addrs: List[str],
    container_rank: int,
    num_slots: int,
    latest_checkpoint: Optional[str] = None,
) -> Iterator[det.ClusterInfo]:
    old_info = det._info._info
    info = make_mock_cluster_info(container_addrs, container_rank, num_slots, latest_checkpoint)
    det._info._info = info
    try:
        yield info
    finally:
        det._info._info = old_info


@contextlib.contextmanager
def set_resources_id_env_var() -> Iterator[None]:
    try:
        os.environ["DET_RESOURCES_ID"] = "resourcesId"
        yield
    finally:
        del os.environ["DET_RESOURCES_ID"]


def parse_args_check(positive_cases: Dict, negative_cases: Dict, parse_func: Callable) -> None:
    for args, exp in positive_cases.items():
        assert exp == parse_func(args.split()), f"test case failed, args = {args}"

    for args, msg in negative_cases.items():
        old = sys.stderr
        fake = io.StringIO()
        sys.stderr = fake
        try:
            try:
                parse_func(args.split())
            except SystemExit:
                # This is expected.
                err = fake.getvalue()
                assert msg in err, f"test case failed, args='{args}' msg='{msg}', stderr='{err}'"
                continue
            raise AssertionError(f"negative test case did not fail: args='{args}'")
        finally:
            sys.stderr = old
