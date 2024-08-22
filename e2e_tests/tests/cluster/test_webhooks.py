import json
import time
import uuid
from typing import Optional

import pytest

from determined.common import api
from determined.common.api import bindings, errors
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
        mode=bindings.v1WebhookMode.WORKSPACE,
        name="",
        workspaceId=None,
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
        regex = r"(.*)cuda(.*)"

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
            mode=bindings.v1WebhookMode.WORKSPACE,
            name="",
            workspaceId=None,
        ),
    )

    default_path = f"/test/path/here/{str(uuid.uuid4())}"
    bindings.post_PostWebhook(
        sess,
        body=bindings.v1Webhook(
            url=f"http://localhost:{port}{default_path}",
            webhookType=bindings.v1WebhookType.DEFAULT,
            triggers=[webhook_trigger],
            mode=bindings.v1WebhookMode.WORKSPACE,
            name="",
            workspaceId=None,
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


def create_default_webhook(sess: api.Session, workspaceId: Optional[int] = None) -> int:
    webhook_trigger = bindings.v1Trigger(
        triggerType=bindings.v1TriggerType.EXPERIMENT_STATE_CHANGE,
        condition={"state": "COMPLETED"},
    )
    webhook_url = "http://localhost"
    res = bindings.post_PostWebhook(
        sess,
        body=bindings.v1Webhook(
            url=webhook_url,
            webhookType=bindings.v1WebhookType.DEFAULT,
            triggers=[webhook_trigger],
            mode=bindings.v1WebhookMode.WORKSPACE,
            name="",
            workspaceId=workspaceId,
        ),
    )
    return res.webhook.id or 0


@pytest.mark.e2e_cpu
def test_webhook_permission() -> None:
    # non-admin should not be able to create global webhook.
    user1_sess = api_utils.user_session()
    with pytest.raises(errors.APIException):
        create_default_webhook(user1_sess)

    # admin should be able to create global webhook.
    admin_sess = api_utils.admin_session()
    global_webhook_id = create_default_webhook(admin_sess)

    # non-admin should be able to view global webhook.
    res = bindings.get_GetWebhooks(user1_sess)
    assert any(w.id == global_webhook_id for w in res.webhooks)

    # user should be able to add webhook to their own workspace
    username = api_utils.get_random_string()
    (user2_sess, _) = api_utils.create_test_user(
        user=bindings.v1User(username=username, active=True, admin=False),
    )
    workspace = bindings.post_PostWorkspace(
        user2_sess,
        body=bindings.v1PostWorkspaceRequest(
            name=f"workspace_aug_{uuid.uuid4().hex[:8]}",
        ),
    ).workspace
    # api_utils.assign_user_role(admin_sess, username, role="Editor", workspace="Uncategorized")
    workspace_webhook_id = create_default_webhook(user2_sess, workspace.id)
    # user should not add workspace to other users' workspace
    with pytest.raises(errors.APIException):
        create_default_webhook(user1_sess, workspace.id)
    # user should be able to get webhook from their own workspace
    res = bindings.get_GetWebhooks(user2_sess)
    assert any(w.id == workspace_webhook_id for w in res.webhooks)
    # user should not be able to get webhook from other users' workspace
    res = bindings.get_GetWebhooks(user1_sess)
    assert not any(w.id == workspace_webhook_id for w in res.webhooks)
    # admin should be able to get all webhooks
    res = bindings.get_GetWebhooks(admin_sess)
    assert any(w.id == workspace_webhook_id for w in res.webhooks)
    assert any(w.id == global_webhook_id for w in res.webhooks)
    # user should not delete webhook from other users' workspace
    with pytest.raises(errors.APIException):
        bindings.delete_DeleteWebhook(user1_sess, id=workspace_webhook_id)
    # non admin should not delete global webhooks
    with pytest.raises(errors.APIException):
        bindings.delete_DeleteWebhook(user1_sess, id=global_webhook_id)
    with pytest.raises(errors.APIException):
        bindings.delete_DeleteWebhook(user2_sess, id=global_webhook_id)
    # admin should be able to delete global webhook
    bindings.delete_DeleteWebhook(admin_sess, id=global_webhook_id)
    # user should be able to delete webhook from their own workspace
    bindings.delete_DeleteWebhook(user2_sess, id=workspace_webhook_id)


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_webhook_rbac() -> None:
    # non-admin should not be able to create global webhook.
    user1_sess = api_utils.user_session()
    with pytest.raises(errors.ForbiddenException):
        create_default_webhook(user1_sess)

    # admin should be able to create global webhook.
    admin_sess = api_utils.admin_session()
    global_webhook_id = create_default_webhook(admin_sess)

    # non-admin should be able to view global webhook.
    res = bindings.get_GetWebhooks(user1_sess)
    assert any(w.id == global_webhook_id for w in res.webhooks)

    # user should be able to add webhook to workspace they have Editor access.
    username = api_utils.get_random_string()
    (user2_sess, _) = api_utils.create_test_user(
        user=bindings.v1User(username=username, active=True, admin=False),
    )
    workspace = bindings.post_PostWorkspace(
        admin_sess,
        body=bindings.v1PostWorkspaceRequest(
            name=f"workspace_aug_{uuid.uuid4().hex[:8]}",
        ),
    ).workspace
    api_utils.assign_user_role(admin_sess, username, role="Editor", workspace=workspace.name)
    workspace_webhook_id = create_default_webhook(user2_sess, workspace.id)
    # user without Editor access should not manage webhook
    with pytest.raises(errors.ForbiddenException):
        create_default_webhook(user1_sess, workspace.id)
    # user should be able to get webhook from workspace they have access to
    res = bindings.get_GetWebhooks(user2_sess)
    assert any(w.id == workspace_webhook_id for w in res.webhooks)
    # user should not be able to get webhook from other users' workspace
    res = bindings.get_GetWebhooks(user1_sess)
    assert not any(w.id == workspace_webhook_id for w in res.webhooks)
    # admin should be able to get all webhooks
    res = bindings.get_GetWebhooks(admin_sess)
    assert any(w.id == workspace_webhook_id for w in res.webhooks)
    assert any(w.id == global_webhook_id for w in res.webhooks)
    # user should not delete webhook from workspace they have no access
    with pytest.raises(errors.ForbiddenException):
        bindings.delete_DeleteWebhook(user1_sess, id=workspace_webhook_id)
    # user should not delete global webhooks
    with pytest.raises(errors.ForbiddenException):
        bindings.delete_DeleteWebhook(user1_sess, id=global_webhook_id)
    with pytest.raises(errors.ForbiddenException):
        bindings.delete_DeleteWebhook(user2_sess, id=global_webhook_id)
    # admin should be able to delete global webhook
    bindings.delete_DeleteWebhook(admin_sess, id=global_webhook_id)
    # user with editor access to workspace should be able to delete it's webhook
    bindings.delete_DeleteWebhook(user2_sess, id=workspace_webhook_id)
