import json
import random
import time
import uuid
from typing import Optional

import pytest

from determined.common import api
from determined.common.api import bindings, errors
from determined.experimental import client
from tests import api_utils
from tests import config as conf
from tests import experiment as exp
from tests.cluster import test_agent_user_group, utils
from tests.experiment import noop


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

    exp_ref = noop.create_experiment(sess)
    exp_ref.wait(interval=0.01)
    assert exp_ref.config
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
                            "text": "✅ " + exp_ref.config["name"] + f" (#{exp_ref.id})",
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
    ws_id = []

    regex = r"executing.*action.*exit.*code.*7"
    if not should_match:
        regex = r"(.*)this should not match(.*)"

    webhook_trigger = bindings.v1Trigger(
        triggerType=bindings.v1TriggerType.TASK_LOG,
        condition={"regex": regex},
    )

    slack_path = f"/test/slack/path/here/{str(uuid.uuid4())}"
    w = bindings.post_PostWebhook(
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
    ws_id.append(w.webhook.id)

    default_path = f"/test/path/here/{str(uuid.uuid4())}"
    w = bindings.post_PostWebhook(
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
    ws_id.append(w.webhook.id)

    workspace = bindings.post_PostWorkspace(
        sess, body=bindings.v1PostWorkspaceRequest(name=f"webhook-test{random.random()}")
    ).workspace
    project = bindings.post_PostProject(
        sess,
        body=bindings.v1PostProjectRequest(
            name=f"webhook-test{random.random()}",
            workspaceId=workspace.id,
        ),
        workspaceId=workspace.id,
    ).project

    specific_path = f"/test/path/here/{str(uuid.uuid4())}"
    w = bindings.post_PostWebhook(
        sess,
        body=bindings.v1Webhook(
            url=f"http://localhost:{port}{specific_path}",
            webhookType=bindings.v1WebhookType.DEFAULT,
            triggers=[webhook_trigger],
            mode=bindings.v1WebhookMode.SPECIFIC,
            name="specific-webhook",
            workspaceId=workspace.id,
        ),
    )
    ws_id.append(w.webhook.id)

    specific_path_unmatch = f"/test/path/here/{str(uuid.uuid4())}"
    w = bindings.post_PostWebhook(
        sess,
        body=bindings.v1Webhook(
            url=f"http://localhost:{port}{specific_path_unmatch}",
            webhookType=bindings.v1WebhookType.DEFAULT,
            triggers=[webhook_trigger],
            mode=bindings.v1WebhookMode.SPECIFIC,
            name=f"webhook-test{random.random()}",
            workspaceId=1,
        ),
    )
    ws_id.append(w.webhook.id)

    config = {"integrations": {"webhooks": {"webhook_name": ["specific-webhook"]}}}
    exp_ref = noop.create_experiment(sess, [noop.Exit(7)], config=config, project_id=project.id)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

    for _ in range(8):
        responses = server.return_responses()
        time.sleep(1)

    responses = server.close_and_return_responses()
    if should_match:
        assert len(responses) >= 3
        # Only need a spot check we get the default / slack responses.
        # Further tested in integrations.
        assert "TASK_LOG" in responses[default_path]
        assert "This log matched the regex" in responses[slack_path]
        assert "TASK_LOG" in responses[specific_path]
        assert specific_path_unmatch not in responses
    else:
        assert default_path not in responses
        assert slack_path not in responses
        assert specific_path not in responses
        assert specific_path_unmatch not in responses

    for i in ws_id:
        bindings.delete_DeleteWebhook(sess, id=i or 0)
    # Delete the project so the workspace can be deleted
    bindings.delete_DeleteProject(sess, id=project.id)
    # Wait for deletion
    time.sleep(0.5)
    test_agent_user_group._delete_workspace_and_check(sess, workspace)


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("isSlack", [True, False])
def test_custom_webhook(isSlack: bool) -> None:
    port = 5009 if isSlack else 5010
    server = utils.WebhookServer(port, allow_dupes=True)
    sess = api_utils.admin_session()
    workspace = bindings.post_PostWorkspace(
        sess, body=bindings.v1PostWorkspaceRequest(name=f"webhook-test{random.random()}")
    ).workspace
    project = bindings.post_PostProject(
        sess,
        body=bindings.v1PostProjectRequest(
            name=f"webhook-test{random.random()}",
            workspaceId=workspace.id,
        ),
        workspaceId=workspace.id,
    ).project

    webhook = bindings.v1Webhook(
        url=f"http://localhost:{port}",
        webhookType=bindings.v1WebhookType.SLACK if isSlack else bindings.v1WebhookType.DEFAULT,
        triggers=[
            bindings.v1Trigger(
                triggerType=bindings.v1TriggerType.CUSTOM,
            )
        ],
        mode=bindings.v1WebhookMode.WORKSPACE,
        name=f"webhook_1{random.random()}",
        workspaceId=workspace.id,
    )
    # custom triggers only work on webhook with mode specific
    with pytest.raises(errors.APIException):
        bindings.post_PostWebhook(sess, body=webhook)
    webhook.mode = bindings.v1WebhookMode.SPECIFIC
    w = bindings.post_PostWebhook(sess, body=webhook).webhook

    experiment_id = exp.create_experiment(
        sess,
        conf.fixtures_path("core_api/11_generic_metrics.yaml"),
        conf.fixtures_path("core_api"),
        [
            "--project_id",
            f"{project.id}",
            "--config",
            f"integrations.webhooks.webhook_name=['{webhook.name}']",
        ],
    )

    # this experiment should not trigger webhook because the name does not match.
    control_exp_id = exp.create_experiment(
        sess,
        conf.fixtures_path("core_api/11_generic_metrics.yaml"),
        conf.fixtures_path("core_api"),
        [
            "--project_id",
            f"{project.id}",
            "--config",
            "integrations.webhooks.webhook_name=['abc']",
        ],
    )

    exp.wait_for_experiment_state(
        sess,
        experiment_id,
        bindings.experimentv1State.COMPLETED,
        max_wait_secs=conf.DEFAULT_MAX_WAIT_SECS,
    )
    exp.wait_for_experiment_state(
        sess,
        control_exp_id,
        bindings.experimentv1State.COMPLETED,
        max_wait_secs=conf.DEFAULT_MAX_WAIT_SECS,
    )

    responses = server.close_and_return_responses()
    assert len(responses) == 1
    assert "end of main" in responses["/"]
    assert "DEBUG" in responses["/"]
    assert str(experiment_id) in responses["/"]

    bindings.delete_DeleteWebhook(sess, id=w.id or 0)
    # Delete the project so the workspace can be deleted
    bindings.delete_DeleteProject(sess, id=project.id)
    # Wait for deletion
    time.sleep(0.5)
    test_agent_user_group._delete_workspace_and_check(sess, workspace)


@pytest.mark.e2e_cpu
def test_specific_webhook() -> None:
    port1 = 5007
    server1 = utils.WebhookServer(port1, allow_dupes=True)
    port2 = 5008
    server2 = utils.WebhookServer(port2, allow_dupes=True)
    sess = api_utils.admin_session()

    workspace = bindings.post_PostWorkspace(
        sess, body=bindings.v1PostWorkspaceRequest(name=f"webhook-test{random.random()}")
    ).workspace
    project = bindings.post_PostProject(
        sess,
        body=bindings.v1PostProjectRequest(
            name=f"webhook-test{random.random()}",
            workspaceId=workspace.id,
        ),
        workspaceId=workspace.id,
    ).project

    webhook_trigger = bindings.v1Trigger(
        triggerType=bindings.v1TriggerType.EXPERIMENT_STATE_CHANGE,
        condition={"state": "COMPLETED"},
    )

    webhook_1 = bindings.v1Webhook(
        url=f"http://localhost:{port1}",
        webhookType=bindings.v1WebhookType.SLACK,
        triggers=[webhook_trigger],
        mode=bindings.v1WebhookMode.SPECIFIC,
        name=f"webhook_1{random.random()}",
        workspaceId=1,
    )

    webhook_2 = bindings.v1Webhook(
        url=f"http://localhost:{port2}",
        webhookType=bindings.v1WebhookType.SLACK,
        triggers=[webhook_trigger],
        mode=bindings.v1WebhookMode.SPECIFIC,
        name="webhook_2",
        workspaceId=workspace.id,
    )

    webhook_res_1 = bindings.post_PostWebhook(sess, body=webhook_1).webhook
    assert webhook_res_1.url == webhook_1.url
    webhook_res_2 = bindings.post_PostWebhook(sess, body=webhook_2).webhook
    assert webhook_res_2.url == webhook_2.url

    config = {"integrations": {"webhooks": {"webhook_id": [webhook_res_1.id, webhook_res_2.id]}}}
    exp_ref = noop.create_experiment(sess, config=config, project_id=project.id)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    responses = server1.close_and_return_responses()
    assert len(responses) == 0
    responses = server2.close_and_return_responses()
    assert len(responses) == 1

    bindings.delete_DeleteWebhook(sess, id=webhook_res_1.id or 0)
    bindings.delete_DeleteWebhook(sess, id=webhook_res_2.id or 0)

    # Delete the project so the workspace can be deleted
    bindings.delete_DeleteProject(sess, id=project.id)
    # Wait for deletion
    time.sleep(0.5)
    test_agent_user_group._delete_workspace_and_check(sess, workspace)


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
    assert res.webhook.id is not None
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

    test_agent_user_group._delete_workspace_and_check(admin_sess, workspace)


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

    test_agent_user_group._delete_workspace_and_check(admin_sess, workspace)


@pytest.mark.e2e_cpu
def test_editing_webhook() -> None:
    port = 5009
    sess = api_utils.admin_session()

    webhook_trigger = bindings.v1Trigger(
        triggerType=bindings.v1TriggerType.TASK_LOG,
        condition={"regex": "test-regex"},
    )

    default_path = f"/test/path/here/{str(uuid.uuid4())}"
    res = bindings.post_PostWebhook(
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
    default_id = res.webhook.id
    assert default_id is not None

    specific_path = f"/test/path/here/{str(uuid.uuid4())}"
    res = bindings.post_PostWebhook(
        sess,
        body=bindings.v1Webhook(
            url=f"http://localhost:{port}{specific_path}",
            webhookType=bindings.v1WebhookType.DEFAULT,
            triggers=[webhook_trigger],
            mode=bindings.v1WebhookMode.SPECIFIC,
            name="specific-webhook2",
            workspaceId=1,
        ),
    )
    specific_id = res.webhook.id
    assert specific_id is not None

    modified_path = f"/test/path/here/{str(uuid.uuid4())}"
    bindings.patch_PatchWebhook(
        sess,
        body=bindings.v1PatchWebhook(url=f"http://localhost:{port}{modified_path}"),
        id=default_id,
    )
    bindings.patch_PatchWebhook(
        sess,
        body=bindings.v1PatchWebhook(url=f"http://localhost:{port}{modified_path}"),
        id=specific_id,
    )

    get_res = bindings.get_GetWebhooks(sess)
    for webhook in get_res.webhooks:
        if webhook.id == default_id or webhook.id == specific_id:
            assert webhook.url == f"http://localhost:{port}{modified_path}"

    bindings.delete_DeleteWebhook(sess, id=default_id)
    bindings.delete_DeleteWebhook(sess, id=specific_id)


@pytest.mark.e2e_cpu
def test_log_pattern_webhook_cache_when_url_is_updated() -> None:
    original_port = 5011
    updated_port = 5012
    original_server = utils.WebhookServer(original_port)
    updated_server = utils.WebhookServer(updated_port)
    sess = api_utils.admin_session()

    regex = r"executing.*action.*exit.*code.*7"

    webhook_trigger = bindings.v1Trigger(
        triggerType=bindings.v1TriggerType.TASK_LOG,
        condition={"regex": regex},
    )

    slack_path = f"/test/slack/path/here/{str(uuid.uuid4())}"
    slack_webhook = bindings.post_PostWebhook(
        sess,
        body=bindings.v1Webhook(
            url=f"http://localhost:{original_port}{slack_path}",
            webhookType=bindings.v1WebhookType.SLACK,
            triggers=[webhook_trigger],
            mode=bindings.v1WebhookMode.WORKSPACE,
            name="",
            workspaceId=None,
        ),
    )
    slack_webhook_id = slack_webhook.webhook.id
    assert slack_webhook_id is not None

    default_path = f"/test/path/here/{str(uuid.uuid4())}"
    default_webhook = bindings.post_PostWebhook(
        sess,
        body=bindings.v1Webhook(
            url=f"http://localhost:{original_port}{default_path}",
            webhookType=bindings.v1WebhookType.DEFAULT,
            triggers=[webhook_trigger],
            mode=bindings.v1WebhookMode.WORKSPACE,
            name="",
            workspaceId=None,
        ),
    )
    default_webhook_id = default_webhook.webhook.id
    assert default_webhook_id is not None

    bindings.patch_PatchWebhook(
        sess,
        body=bindings.v1PatchWebhook(url=f"http://localhost:{updated_port}{slack_path}"),
        id=slack_webhook_id,
    )
    bindings.patch_PatchWebhook(
        sess,
        body=bindings.v1PatchWebhook(url=f"http://localhost:{updated_port}{default_path}"),
        id=default_webhook_id,
    )

    exp_ref = noop.create_experiment(sess, [noop.Exit(7)])
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

    time.sleep(10)
    responses = original_server.close_and_return_responses()
    assert len(responses) == 0

    for _ in range(10):
        responses = updated_server.return_responses()
        if default_path in responses and slack_path in responses:
            break
        time.sleep(1)

    responses = updated_server.close_and_return_responses()
    assert len(responses) >= 2
    # Only need a spot check we get the default / slack responses.
    # Further tested in integrations.
    assert "TASK_LOG" in responses[default_path]
    assert "This log matched the regex" in responses[slack_path]

    bindings.delete_DeleteWebhook(sess, id=slack_webhook_id or 0)
    bindings.delete_DeleteWebhook(sess, id=default_webhook_id or 0)
