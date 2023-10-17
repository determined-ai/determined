import json

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
    sess = api_utils.determined_test_session(admin=True)

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
        conf.fixtures_path("no_op/single-one-short-step.yaml"), conf.fixtures_path("no_op")
    )

    exp.wait_for_experiment_state(
        experiment_id,
        bindings.experimentv1State.COMPLETED,
        max_wait_secs=conf.DEFAULT_MAX_WAIT_SECS,
    )
    exp_config = exp.experiment_config_json(experiment_id)
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
