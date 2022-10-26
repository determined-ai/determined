import json
import threading
from http.server import HTTPServer, SimpleHTTPRequestHandler

import pytest

from determined.common import api
from determined.common.api import authentication, bindings
from tests import config as conf
from tests import experiment as exp
from tests.cluster.test_users import ADMIN_CREDENTIALS

# global variable to store the webhook request
request_to_webhook_endpoint = {}

# global state to handle server termination
keep_server_running = True

SERVER_PORT = 5005


class WebhookRequestHandler(SimpleHTTPRequestHandler):
    def do_POST(self) -> None:
        global request_to_webhook_endpoint
        global keep_server_running
        content_length = int(self.headers.get("content-length"))
        request_body = self.rfile.read(content_length)
        request_to_webhook_endpoint = json.loads(request_body)
        self.send_response(200, "Success")
        self.end_headers()
        self.wfile.write("".encode("utf-8"))

        # terminate Server
        keep_server_running = False


def run_server(server_class=HTTPServer, handler_class=WebhookRequestHandler) -> None:
    global keep_server_running
    server_address = ("", SERVER_PORT)
    http_server = server_class(server_address, handler_class)
    while keep_server_running:
        http_server.handle_request()


@pytest.mark.e2e_cpu
def test_slack_webhook() -> None:
    global request_to_webhook_endpoint
    server_thread = threading.Thread(target=run_server, daemon=True)
    server_thread.start()
    sess = exp.determined_test_session(admin=True)

    webhook_trigger = bindings.v1Trigger(
        triggerType=bindings.v1TriggerType.TRIGGER_TYPE_EXPERIMENT_STATE_CHANGE,
        condition={"state": "COMPLETED"},
    )

    webhook_request = bindings.v1Webhook(
        url=f"http://localhost:{SERVER_PORT}",
        webhookType=bindings.v1WebhookType.WEBHOOK_TYPE_SLACK,
        triggers=[webhook_trigger],
    )

    result = bindings.post_PostWebhook(sess, body=webhook_request)
    assert result.webhook.url == webhook_request.url

    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-one-short-step.yaml"), conf.fixtures_path("no_op")
    )

    exp.wait_for_experiment_state(
        experiment_id,
        bindings.determinedexperimentv1State.STATE_COMPLETED,
        max_wait_secs=conf.DEFAULT_MAX_WAIT_SECS,
    )
    exp_config = exp.experiment_config_json(experiment_id)

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
    server_thread.join()
    assert expected_payload == request_to_webhook_endpoint
