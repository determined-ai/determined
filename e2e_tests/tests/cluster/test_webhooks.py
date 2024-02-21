import json
import time
import uuid

import pytest

from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import experiment as exp
from tests.cluster import utils


@pytest.mark.e2e_cpu
def test_slack_webhook() -> None:
    port = 5005
    server = utils.WebhookServer(port, allow_dupes=True)
    sess = api_utils.admin_session()

    webhook_trigger = bindings.v1Trigger(
        triggerType=bindings.v1TriggerType.EXPERIMENT_STATE_CHANGE,
        condition={"state": "COMPLETED"},
    )

    webhook_request = bindings.v1Webhook(
        url=f"http://localhost:{port}",
        webhookType=bindings.v1WebhookType.SLACK,
        triggers=[webhook_trigger],
    )

    result = bindings.post_PostWebhook(sess, body=webhook_request)
    assert result.webhook.url == webhook_request.url

    experiment_id = exp.create_experiment(
        sess, conf.fixtures_path("no_op/single-one-short-step.yaml"), conf.fixtures_path("no_op")
    )

    exp.wait_for_experiment_state(
        sess,
        experiment_id,
        bindings.experimentv1State.COMPLETED,
        max_wait_secs=conf.DEFAULT_MAX_WAIT_SECS,
    )
    exp_config = exp.experiment_config_json(sess, experiment_id)
    expected_field = {"type": "mrkdwn", "text": "*Status*: Completed"}
    expected_payload = {
        "blocks": [
            {
                "type": "section",
                "text": {"type": "plain_text", "text": "Your experiment completed successfully 🎉"},
            }
        ],
        "attachments": [
            {
                "color": "#13B670",
                "blocks": [
                    {
                        "type": "section",
                        "text": {
                            "type": "mrkdwn",
                            "text": "✅ " + exp_config["name"] + f" (#{experiment_id})",
                        },
                        "fields": [
                            {"type": "mrkdwn", "text": "*Status*: Completed"},
                            {"type": "mrkdwn", "text": "*Duration*: 0h 0min"},
                        ],
                    }
                ],
            }
        ],
    }
    expected_color = "#13B670"

    responses = server.close_and_return_responses()
    assert len(responses) == 1
    response = json.loads(responses["/"])

    assert expected_payload["blocks"] == response["blocks"]
    assert expected_color == response["attachments"][0]["color"]
    assert expected_field == response["attachments"][0]["blocks"][0]["fields"][0]


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("should_match", [True, False])
def test_log_pattern_send_webhook(should_match: bool) -> None:
    port = 5006
    server = utils.WebhookServer(port)
    sess = api_utils.admin_session()

    regex = r"assert 0 <= self\.metrics_sigma"
    if not should_match:
        regex = r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b"

    webhook_trigger = bindings.v1Trigger(
        triggerType=bindings.v1TriggerType.TASK_LOG,
        condition={"regex": regex},
    )

    slack_path = f"/test/slack/path/here/{str(uuid.uuid4())}"
    bindings.post_PostWebhook(
        sess,
        body=bindings.v1Webhook(
            url=f"http://localhost:{port}{slack_path}",
            webhookType=bindings.v1WebhookType.SLACK,
            triggers=[webhook_trigger],
        ),
    )

    default_path = f"/test/path/here/{str(uuid.uuid4())}"
    bindings.post_PostWebhook(
        sess,
        body=bindings.v1Webhook(
            url=f"http://localhost:{port}{default_path}",
            webhookType=bindings.v1WebhookType.DEFAULT,
            triggers=[webhook_trigger],
        ),
    )

    exp_id = exp.create_experiment(
        sess,
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--config", "hyperparameters.metrics_sigma=-1.0"],
    )
    exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.ERROR)

    for _ in range(10):
        responses = server.return_responses()
        if default_path in responses and slack_path in responses:
            break
        time.sleep(1)

    responses = server.close_and_return_responses()
    if should_match:
        assert len(responses) >= 2
        # Only need a spot check we get the default / slack responses.
        # Further tested in integrations.
        assert "TASK_LOG" in responses[default_path]
        assert "This log matched the regex" in responses[slack_path]
    else:
        assert default_path not in responses
        assert slack_path not in responses
