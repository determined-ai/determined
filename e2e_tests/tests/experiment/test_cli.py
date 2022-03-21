from argparse import Namespace
from pathlib import Path

import pytest

import determined.cli.experiment as experiment_cli
from determined.common.api import bindings, errors
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_experiment_cli() -> None:
    exp_id = exp.run_basic_test(
        conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"),
        conf.tutorials_path("mnist_pytorch"),
        None,
    )
    master_url = conf.make_master_url()
    sess = exp.test_session()

    def get_experiment(exp_id: int) -> bindings.v1Experiment:
        return bindings.get_GetExperiment(sess, experimentId=exp_id).experiment

    args = Namespace(
        all=True,
        csv=False,
        description="My Description",
        experiment_id=exp_id,
        experiment_ids=str(exp_id),
        json=False,
        label="tested",
        limit=1,
        master=master_url,
        max_slots=10,
        metrics=False,
        name="My Name",
        offset=0,
        outdir=Path("/tmp"),
        output_dir=Path("/tmp"),
        priority=11,
        save_experiment_best=3,
        save_trial_best=2,
        save_trial_latest=1,
        user="determined",
        weight=12,
        yes=True,
    )

    # these commands raise exceptions because the experiment already completed
    with pytest.raises(errors.APIException):
        experiment_cli.activate(args)
    with pytest.raises(errors.APIException):
        experiment_cli.pause(args)

    # commands which set/unset experiment attributes
    experiment_cli.archive(args)
    assert get_experiment(exp_id).archived
    experiment_cli.unarchive(args)
    assert not get_experiment(exp_id).archived
    experiment_cli.set_description(args)
    assert get_experiment(exp_id).description == "My Description"
    experiment_cli.set_name(args)
    assert get_experiment(exp_id).name == "My Name"
    experiment_cli.add_label(args)
    assert get_experiment(exp_id).labels == ["tested"]
    experiment_cli.remove_label(args)
    assert get_experiment(exp_id).labels == []
    experiment_cli.set_max_slots(args)
    experiment_cli.set_priority(args)
    experiment_cli.set_weight(args)
    experiment_cli.set_gc_policy(args)
    final_config = bindings.get_GetExperiment(sess, experimentId=exp_id).config
    assert final_config["resources"]["max_slots"] == args.max_slots
    assert final_config["resources"]["priority"] == args.priority
    assert final_config["resources"]["weight"] == args.weight
    assert final_config["checkpoint_storage"]["save_experiment_best"] == 3
    assert final_config["checkpoint_storage"]["save_trial_best"] == 2
    assert final_config["checkpoint_storage"]["save_trial_latest"] == 1

    # commands which write to standard output
    experiment_cli.config(args)
    experiment_cli.download_model_def(args)

    # commands tested in e2e_tests/tests/experiment/experiment.py
    # experiment_cli.describe(args)
    # experiment_cli.list_experiments(args)
    # experiment_cli.list_trials(args)

    # commands which change state but experiment already completed
    experiment_cli.cancel(args)
    experiment_cli.kill_experiment(args)

    # delete test experiment and verify it is no longer listed
    experiment_cli.delete_experiment(args)
    with pytest.raises(errors.NotFoundException):
        get_experiment(exp_id)
