import datetime

import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.performance  # type: ignore
def test_mask_rcnn_64_slots() -> None:
    experiment_id = exp.run_basic_test(
        conf.experimental_path("trial/FasterRCNN_tp/64-gpus.yaml"),
        conf.experimental_path("trial/FasterRCNN_tp/"),
        1,
        max_wait_secs=5 * 60 * 60,
    )

    validation_metric_name = "mAP(bbox)/IoU=0.5:0.95"
    validation_metric = exp.get_validation_metric_from_last_step(
        experiment_id, 0, validation_metric_name
    )
    durations = exp.get_experiment_durations(experiment_id, 0)
    wait_for_agents_time = (
        durations.experiment_duration
        - durations.training_duration
        - durations.validation_duration
        - durations.checkpoint_duration
    )

    print(validation_metric_name, validation_metric)
    print(durations)
    print(f"wait for agents duration: {wait_for_agents_time}")

    assert validation_metric > 0.375
    assert durations.training_duration < datetime.timedelta(hours=2, minutes=45)
    assert durations.validation_duration < datetime.timedelta(hours=1, minutes=15)
