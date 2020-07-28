import boto3
import datetime
import os

from dateutil.tz import tzutc
from slack import WebClient
from slack.errors import SlackApiError
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


if __name__ == "__main__":
    client = boto3.client("cloudformation")
    response = client.describe_stacks()
    timeout = datetime.timedelta(hours=6)
    slack_token = os.environ["SLACK_API_TOKEN"]
    print(slack_token)
    client = WebClient(token=slack_token)

    targetStacks = {"nightly": ["nightly-test"], "e2e-gpu": [], "parallel": []}

    for each in response["Stacks"]:
        check_conditions(each, targetStacks, timeout)

    print("about to list all qualifying stacks")
    for k, lists in targetStacks.items():
        for stack in lists:
            print("stack: " + stack)
            # os.system(f"det-deploy aws down --cluster-id {stack}")
            try:
                print("sending message")
                response = client.chat_postMessage(
                    channel="D015DMR4XKR",
                    text="tearing down "+ stack,
                    username = "alfred"
                )
            except SlackApiError as e :
                print(e.response["error"])
