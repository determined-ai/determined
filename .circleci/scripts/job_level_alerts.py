import json
import logging
import os
import time
from typing import Set

import requests

workflows_to_skip = {
    # Skip test-e2e-longrunning for testing on feature branches since it won't complete on them.
    "test-e2e-longrunning",
    "send-alerts",
    # Skip nightly tests as a temporary measure to bring this ci check back.
    # https://hpe-aiatscale.slack.com/archives/C04C9JXB1C2/p1720448458815989
    "nightly",
}


def send_alert(job_name: str, pipeline_number: str, workflow_id: str, job_number: str) -> None:
    job_url = f"https://app.circleci.com/pipelines/github/determined-ai/determined/{pipeline_number}/workflows/{workflow_id}/jobs/{job_number}"  # noqa: E501

    # TODO(RM-252) mention the team who owns the test.
    slack_message = f"{job_name} failed on main, {job_url}"
    print(f"sending slack message: {slack_message}")

    r = requests.post(
        os.environ["SLACK_WEBHOOK"],
        headers={"Content-Type": "application/json"},
        data=json.dumps(
            {
                "text": slack_message,
            }
        ),
    )
    assert r.content == b"ok", r.content


def send_alerts_for_failed_jobs(sent_alerts: Set[str]) -> bool:
    pipeline_id = os.environ["CIRCLE_PIPELINE_ID"]
    workflows = requests.get(f"https://circleci.com/api/v2/pipeline/{pipeline_id}/workflow").json()
    workflows_are_running = False
    for w in workflows["items"]:
        if w["name"] in workflows_to_skip:
            continue

        workflow_id = w["id"]
        if not workflows_are_running and w["stopped_at"] is None:
            print(f"waiting for at least workflow {w['name']} to finish")
            workflows_are_running = True

        jobs = requests.get(f"https://circleci.com/api/v2/workflow/{workflow_id}/job").json()
        for j in jobs["items"]:
            job_name = j["name"]
            if workflow_id + job_name not in sent_alerts and j["status"] == "failed":
                send_alert(job_name, w["pipeline_number"], workflow_id, j["job_number"])
                sent_alerts.add(workflow_id + job_name)

    return workflows_are_running


def main() -> None:
    failure_count = 0
    sent_alerts: Set[str] = set()
    while failure_count < 20:
        try:
            print("Checking circleci API for jobs")
            still_workflows_in_progress = send_alerts_for_failed_jobs(sent_alerts)
            if not still_workflows_in_progress:
                print("all workflows complete, ending")
                return
            failure_count = 0
        except Exception as e:
            logging.critical(e, exc_info=True)
            failure_count += 1

        time.sleep(15)


if __name__ == "__main__":
    main()
