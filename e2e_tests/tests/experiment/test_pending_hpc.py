import pytest

from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


# Test only works on resource pool with 1 node 8 CPU slots.
# Queries the determined master for resource pool information to determine if
# resource pool is suitable for this test.
def skip_if_not_suitable_resource_pool() -> None:
    session = api_utils.user_session()
    rps = bindings.get_GetResourcePools(session)
    assert rps.resourcePools and len(rps.resourcePools) > 0, "missing resource pool"
    if (
        len(rps.resourcePools) != 1
        or rps.resourcePools[0].slotType != bindings.devicev1Type.CPU
        or rps.resourcePools[0].slotsAvailable != 8
    ):
        errorMessage = "required config: 1 resource pool with 1 node, with only 8 slots."
        pytest.skip(errorMessage)


@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@api_utils.skipif_not_hpc()
def test_hpc_job_pending_reason() -> None:
    skip_if_not_suitable_resource_pool()
    sess = api_utils.user_session()
    detobj = client.Determined._from_session(sess)

    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_slots_per_trial(config, 1)
    config = conf.set_profiling_enabled(config)
    config = conf.set_entrypoint(
        config, "python3 -m determined.launch.torch_distributed python3 train.py"
    )
    config["max_restarts"] = 0

    # The experiment will request 6 CPUs
    config.setdefault("slurm", {})
    config["slurm"]["slots_per_node"] = 6
    config.setdefault("pbs", {})
    config["pbs"]["slots_per_node"] = 6

    running_exp = detobj.create_experiment(config, conf.fixtures_path("mnist_pytorch"))
    print(f"Created running experiment {running_exp.id}")
    exp.wait_for_experiment_state(sess, running_exp.id, bindings.experimentv1State.RUNNING)

    # Launch another experiment requesting 6 CPUs
    pending_exp = detobj.create_experiment(config, conf.fixtures_path("mnist_pytorch"))
    print(f"Created pending experiment {pending_exp.id}")

    exp.wait_for_experiment_state(sess, pending_exp.id, bindings.experimentv1State.QUEUED)

    # Wait for the second experiment to show it is pending in its logs.
    pattern = "HPC job waiting to be scheduled"
    logs = []
    for log in pending_exp.await_first_trial().iter_logs(follow=True):
        if pattern in log:
            break
        logs.append(log)
    else:
        text = "".join(logs)
        raise ValueError(
            f"did not find '{pattern}' in logs:\n-- BEGIN TEXT --\n{text}-- END TEXT --"
        )

    # Release resources, letting the pending experiment onto the cluster.
    running_exp.kill()
    running_exp.wait(interval=0.01)

    # Make sure the second experiment will start running after the first experiment
    # releases the CPUs
    exp.wait_for_experiment_state(sess, pending_exp.id, bindings.experimentv1State.RUNNING)

    # Don't care if the experiment finishes.
    pending_exp.kill()
    pending_exp.wait(interval=0.01)
