from typing import List

import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.parallel
def test_pytorch_gradient_aggregation() -> None:
    config = conf.load_config(conf.fixtures_path("pytorch_identity/distributed.yaml"))

    exp_id = exp.run_basic_test_with_temp_config(config, conf.fixtures_path("pytorch_identity"), 1)
    trials = exp.experiment_trials(exp_id)
    assert len(trials) == 1
    workloads = exp.workloads_with_validation(trials[0].workloads)
    actual_weights = []
    for wl in workloads:
        if wl.metrics:
            actual_weights.append(wl.metrics.avgMetrics["weight"])

    # independently compute expected metrics
    batch_size = 4
    epoch_size = 64
    num_epochs = 3
    batches = [
        (v[:], v[:])
        for v in (
            [x * 0.1 + 1.0 for x in range(y, y + batch_size)]
            for y in (z % epoch_size for z in range(0, epoch_size * num_epochs, batch_size))
        )
    ]

    lr = 0.001

    def compute_expected_weight(data: List[float], label: List[float], w: float) -> float:
        n = len(data)
        expected_step = 2.0 * lr * sum((d * (l - d * w) for d, l in zip(data, label))) / n
        return w + expected_step

    expected_weights = []
    weight = 0.0
    data: List[float] = []
    label: List[float] = []
    for i, batch in enumerate(batches):
        if i % 2 == 0:
            # for even-numbered batches the optimizer step is a no-op:
            # the weights don't change
            data, label = batch
        else:
            additional_data, additional_label = batch
            data += additional_data
            label += additional_label
            weight = compute_expected_weight(data, label, weight)
        expected_weights.append(weight)

    assert actual_weights == pytest.approx(
        expected_weights
    ), f"{actual_weights} != {expected_weights}"
