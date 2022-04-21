import contextlib
import io
import os
import sys

import determined as det
from typing import List, Iterator, Callable, Dict
from tests.experiment import utils


def make_mock_cluster_info(
        container_addrs: List[str], container_rank: int, num_slots: int
) -> det.ClusterInfo:
    config = utils.make_default_exp_config({}, 100, "loss", None)
    trial_info_mock = det.TrialInfo(
        trial_id=1,
        experiment_id=1,
        trial_seed=0,
        hparams={},
        config=config,
        latest_batch=0,
        trial_run_id=0,
        debug=False,
        unique_port_offset=0,
        inter_node_network_interface=None,
    )
    rendezvous_info_mock = det.RendezvousInfo(
        container_addrs=container_addrs, container_rank=container_rank
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
    )
    return cluster_info_mock


@contextlib.contextmanager
def set_resources_id_env_var() -> Iterator[None]:
    try:
        os.environ["DET_RESOURCES_ID"] = "resourcesId"
        yield
    finally:
        del os.environ["DET_RESOURCES_ID"]


def test_parse_args(positive_cases: Dict[str, List], negative_cases: Dict[str, List], parse_func: Callable):
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
