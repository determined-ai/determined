import os
import requests
import time

workflows_to_skip = {
    # Skip test-e2e-longrunning for testing on feature branches since it won't complete on them.
    "test-e2e-longrunning",
    "send-alerts",
}

sent_alerts = {}

# To test locally.
# export CIRCLE_PIPELINE_ID='ca8f23cf-f9a9-4a72-ba5e-e78bfe501dda'
def send_alert(job_name: str, pipeline_number: str, workflow_id: str, job_number: str):
    job_url = f"https://app.circleci.com/pipelines/github/determined-ai/determined/{pipeline_number}/workflows/{workflow_id}/jobs/{job_number}"

    slack_message = f"{job_name} failed on main, {job_url}\nowning team: @TODO"
    print(slack_message)

def send_alerts_for_failed_jobs(sent_alerts):
    pipeline_id = os.environ["CIRCLE_PIPELINE_ID"]
    workflows = requests.get(f"https://circleci.com/api/v2/pipeline/{pipeline_id}/workflow").json()
    workflows_are_running = False
    for w in workflows["items"]:
        if w["name"] in workflows_to_skip:
            continue

        workflow_id = w["id"]
        if not workflows_are_running and w["status"] == "running":
            print(f"waiting for workflow {w['name']} to finish")
            workflows_are_running = True

        jobs = requests.get(f"https://circleci.com/api/v2/workflow/{workflow_id}/job").json()
        for j in jobs["items"]:
            job_name = j["name"]
            if workflow_id + job_name not in sent_alerts and j["status"] == "failed":
                send_alert(job_name, w["pipeline_number"], workflow_id, j["job_number"])
                sent_alerts[workflow_id + job_name] = True

    return workflows_are_running

def main():
    failure_count = 0
    sent_alerts = {}
    while failure_count < 5: # TODO make this higher
        try:
            print("Checking circleci API for jobs")
            still_workflows_in_progress = send_alerts_for_failed_jobs(sent_alerts)
            if not still_workflows_in_progress:
                print("all workflows complete, ending")
                return
        except Exception as e:
            print(f"failed to read circleci state and send alerts: {e}")
            failure_count += 1

        time.sleep(15)


if __name__ == "__main__":
    main()


#api_url = f'https://circleci.com/api/v2/project/github/{username}/{project}/workflow/{workflow}/job'
