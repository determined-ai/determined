from unittest import mock

import pytest

from determined import core
from determined.common import api


@mock.patch("determined.common.api.bindings.post_ReportTrialMetrics")
def test_metrics_report(mock_post_metrics: mock.MagicMock) -> None:
    trial_id = 1
    master_url = "http://test_master:8080"
    session = api.Session(master=master_url, username="user", token="token", cert=None)
    metrics = [
        {"loss": 0.10, "accuracy": 0.90},
        {"loss": 0.20, "accuracy": 0.91},
        {"loss": 0.30, "accuracy": 0.92},
        {"loss": 0.40, "accuracy": 0.93},
        {"loss": 0.50, "accuracy": 0.94},
    ]
    metrics_context = core.MetricsContext(session=session, trial_id=trial_id, run_id=1)

    metrics_context.start()
    for idx, metric in enumerate(metrics):
        metrics_context.report(group="training", metrics=metric, steps_completed=idx + 1)
    metrics_context.close()

    assert mock_post_metrics.call_count == len(metrics)
    for idx, metric in enumerate(metrics):
        call = mock_post_metrics.call_args_list[idx]
        assert call.kwargs["metrics_trialId"] == trial_id

        req_body = call.kwargs["body"]
        assert req_body.group == "training"

        req_metrics = req_body.metrics
        assert req_metrics.metrics.avgMetrics == metric
        assert req_metrics.trialId == trial_id
        assert req_metrics.trialRunId == 1
        assert req_metrics.stepsCompleted == idx + 1


@mock.patch("determined.common.api.bindings.post_ReportTrialMetrics")
def test_metrics_report_raises_exception(mock_post_metrics: mock.MagicMock) -> None:
    trial_id = 1
    master_url = "http://test_master:8080"
    session = api.Session(master=master_url, username="user", token="token", cert=None)

    mock_post_metrics.side_effect = ValueError("Exception in metrics reporting")

    metrics_context = core.MetricsContext(session=session, trial_id=trial_id, run_id=1)

    metrics_context.start()
    with pytest.raises(ValueError, match="Exception in metrics reporting"):
        metrics_context.report(
            group="training", metrics={"loss": 0.10, "accuracy": 0.90}, steps_completed=1
        )
        metrics_context.close()

    assert mock_post_metrics.call_count == 1
