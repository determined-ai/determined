import json
import threading
from http.server import HTTPServer, SimpleHTTPRequestHandler

import pytest

from determined.common import api
from determined.common.api import authentication, bindings
from tests import config as conf
from tests import experiment as exp
from tests.cluster.test_users import ADMIN_CREDENTIALS

request_to_webhook_endpoint = {}
running = True
SERVER_PORT = 5005


class RequestHandler(SimpleHTTPRequestHandler):
    def do_POST(self):
        global request_to_webhook_endpoint
        global running
        content_length = int(self.headers.get("content-length"))
        request_body = self.rfile.read(content_length)
        request_to_webhook_endpoint = json.loads(request_body)
        self.send_response(200, "Success")
        self.end_headers()
        self.wfile.write("".encode("utf-8"))

        # Terminate Server
        running = False


def run_server(server_class=HTTPServer, handler_class=RequestHandler):
    global running
    server_address = ("", SERVER_PORT)
    http_server = server_class(server_address, handler_class)
    while running:
        http_server.handle_request()


@pytest.mark.e2e_cpu
def test_slack_webhook() -> None:
    global request_to_webhook_endpoint
    server_thread = threading.Thread(target=run_server, daemon=True)
    server_thread.start()
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
                            "text": "✅ Experiment (secondly-skilled-shiner) (#126)",
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
    master_url = conf.make_master_url()
    admin_auth = authentication.Authentication(
        master_url, ADMIN_CREDENTIALS.username, ADMIN_CREDENTIALS.password, try_reauth=True
    )
    sess = api.Session(master_url, ADMIN_CREDENTIALS.username, admin_auth, None)

    webhook_trigger_data = {
        "triggerType": bindings.v1TriggerType.TRIGGER_TYPE_EXPERIMENT_STATE_CHANGE,
        "condition": {"state": "COMPLETED"},
    }
    webhook_trigger = bindings.v1Trigger(**webhook_trigger_data)
    webhook = {
        "url": f"http://localhost:{SERVER_PORT}",
        "webhookType": bindings.v1WebhookType.WEBHOOK_TYPE_SLACK,
        "triggers": [webhook_trigger],
    }
    webhook_request = bindings.v1Webhook(**webhook)

    result = bindings.post_PostWebhook(sess, body=webhook_request)
    assert result.webhook.url == webhook["url"]

    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-one-short-step.yaml"), conf.fixtures_path("no_op")
    )

    exp.wait_for_experiment_state(
        experiment_id,
        bindings.determinedexperimentv1State.STATE_COMPLETED,
        max_wait_secs=conf.DEFAULT_MAX_WAIT_SECS,
    )
    server_thread.join()
    expected_fields = expected_payload["attachments"][0]["blocks"][0]["fields"]
    assert expected_payload["blocks"] == request_to_webhook_endpoint["blocks"]
    assert (
        expected_payload["attachments"][0]["color"]
        == request_to_webhook_endpoint["attachments"][0]["color"]
    )
    assert expected_fields == request_to_webhook_endpoint["attachments"][0]["blocks"][0]["fields"]
