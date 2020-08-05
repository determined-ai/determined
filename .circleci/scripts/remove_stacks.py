import boto3
import datetime
import os
import json
import requests

from dateutil.tz import tzutc
from typing import Any, Dict, List


def check_conditions(
        stack: Dict[str, Any],
        target_stacks: Dict[str, List[Any]],
        timeout: datetime.timedelta,
) -> None:
    current_time = datetime.datetime.now(tz=tzutc())

    for k in target_stacks:
        if k in stack["StackName"] and current_time - stack["CreationTime"] > timeout:
            target_stacks[k].append(stack["StackName"])


def send_message(data: Dict[str, str], hook):
    response = requests.post(hook, data=json.dumps(data), headers={'Content-Type': 'application/json'})
    if response.status_code != 200:
        if response.text:
            raise ValueError(
                f"Response to Slack Request returned error {response.status_code} with message: {response.text}"
            )
        else:
            raise ValueError(f"Response to Slack Request returned error {response.status_code}")


if __name__ == "__main__":
    client = boto3.client("cloudformation")
    response = client.describe_stacks()
    timeout = datetime.timedelta(hours=6)

    targetStacks = {"nightly": ["nightly-test"], "e2e-gpu": [], "parallel": []}

    for each in response["Stacks"]:
        check_conditions(each, targetStacks, timeout)

    deleted_stacks = ""
    print(os.getenv("SLACK_DELETION_WEBHOOK"))

    for k, lists in targetStacks.items():
        for stack in lists:
            # print("stack: " + stack)
            # os.system(f"det-deploy aws down --cluster-id {stack}")
            deleted_stacks += f"•`{stack}`\n"

    print(deleted_stacks)

    # send_message({'text': '<!channel>'}, my_hook)
    # payload = {"text": f"*The following stacks have been terminated*\n {deleted_stacks}"}
    # send_message(payload, my_hook)

