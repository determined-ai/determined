import sys
import time
from typing import Any, Dict, List, Optional, Tuple

import boto3
import tqdm
from botocore.exceptions import ClientError, WaiterError

from determined.deploy.aws import constants

# Try waiting for stack to delete this many times. We break up the waiting so the delete job
# will not fail CI.
NUM_WAITS = 5

DELETE_MASTER_IGNORE_ERRORS = (
    "IncorrectInstanceState",
    "InvalidInstanceID.NotFound",
)


class NoStackOutputError(Exception):
    pass


def get_user(boto3_session: boto3.session.Session) -> str:
    sts = boto3_session.client("sts")
    response = sts.get_caller_identity()
    arn = response["Arn"]
    assert isinstance(arn, str), f"expected a string Arn but got {arn}"
    return arn.split("/")[-1]


def stop_master(master_id: str, boto3_session: boto3.session.Session, delete: bool = False) -> None:
    ec2 = boto3_session.client("ec2")
    waiter = ec2.get_waiter("instance_stopped")
    try:
        ec2.stop_instances(InstanceIds=[master_id])
    except ClientError as ex:
        if delete:
            error_code = ex.response.get("Error", {}).get("Code")
            if error_code in DELETE_MASTER_IGNORE_ERRORS:
                print(
                    f"Failed to stop Master Instance: {error_code}. "
                    "This error is ignored as the instance is going to be deleted."
                )
                return

        raise

    ec2.modify_instance_attribute(
        Attribute="disableApiTermination", Value="false", InstanceId=master_id
    )

    for n in range(NUM_WAITS):
        print("Waiting For Master Instance To Stop")
        try:
            waiter.wait(InstanceIds=[master_id], WaiterConfig={"Delay": 10})
            break
        except WaiterError as e:
            if n == NUM_WAITS - 1:
                raise e

    print("Master Instance Stopped")


def delete(stack_name: str, boto3_session: boto3.session.Session) -> None:
    ec2 = boto3_session.client("ec2")

    # Check if we have stack outputs containing ec2 instance and
    # bucket information, if not, just remove the stack.
    try:
        stack_output = get_output(stack_name, boto3_session)
    except NoStackOutputError:
        print(
            f"Stack {stack_name} is in inconsistent state. "
            "This error is ignored as stack is going to be deleted."
        )
        delete_stack(stack_name, boto3_session)
        return

    # First, shut down the master so no new agents are started.
    master_id = stack_output[constants.cloudformation.MASTER_ID]
    describe_instance_response = ec2.describe_instances(
        Filters=[{"Name": "instance-id", "Values": [master_id]}],
    )

    if describe_instance_response["Reservations"]:
        print("Stopping Master Instance")
        stop_master(master_id, boto3_session, delete=True)

    # Second, terminate the agents so nothing can write to the checkpoint bucket. We create agent
    # instances outside of cloudformation, so we have to manually terminate them.
    if stack_uses_spot(stack_name, boto3_session):
        print("Terminating Running Agents and Pending Spot Requests")
        clean_up_spot(stack_name, boto3_session)
        print("Agents and Spot Requests Terminated")
    else:
        print("Terminating Running Agents")
        terminate_running_agents(
            stack_output[constants.cloudformation.AGENT_TAG_NAME], boto3_session
        )
        print("Agents Terminated")

    # Third, empty the bucket that was created for this stack.
    bucket_name = get_output(stack_name, boto3_session).get(
        constants.cloudformation.CHECKPOINT_BUCKET
    )
    if bucket_name:
        print("Emptying Checkpoint Bucket")
        empty_bucket(bucket_name, boto3_session)
        print("Checkpoint Bucket Empty")

    delete_stack(stack_name, boto3_session)


def get_tags(stack_name: str, boto3_session: boto3.session.Session) -> Dict[Any, Any]:
    cfn = boto3_session.client("cloudformation")

    print(f"Retrieving tags for CloudFormation Stack ({stack_name})")

    description = cfn.describe_stacks(StackName=stack_name)
    stack = description["Stacks"][0]
    return {x["Key"]: x["Value"] for x in stack["Tags"]}


# Cloudformation
def stack_exists(stack_name: str, boto3_session: boto3.session.Session) -> bool:
    cfn = boto3_session.client("cloudformation")

    print(f"Checking if the CloudFormation Stack ({stack_name}) exists:", end=" ")

    try:
        cfn.describe_stacks(StackName=stack_name)
    except ClientError:
        return False

    return True


def delete_stack(stack_name: str, boto3_session: boto3.session.Session) -> None:
    cfn = boto3_session.client("cloudformation")
    delete_waiter = cfn.get_waiter("stack_delete_complete")

    if stack_exists(stack_name, boto3_session):
        print(
            f"True - Deleting stack {stack_name}. This may take a few minutes... "
            f"Check the CloudFormation Console for updates"
        )

    else:
        print(f"False. {stack_name} does not exist")
    cfn.delete_stack(StackName=stack_name)
    delete_waiter.wait(StackName=stack_name, WaiterConfig={"Delay": 10})


def update_stack(
    stack_name: str,
    template_body: str,
    boto3_session: boto3.session.Session,
    deployment_type: str,
    parameters: Optional[List] = None,
    update_terminate_agents: bool = True,
) -> None:
    cfn = boto3_session.client("cloudformation")
    ec2 = boto3_session.client("ec2")
    update_waiter = cfn.get_waiter("stack_update_complete")

    print(
        f"Updating stack {stack_name}. This may take a few minutes... "
        f"Check the CloudFormation Console for updates"
    )
    stack_output = get_output(stack_name, boto3_session)

    if update_terminate_agents:
        if stack_uses_spot(stack_name, boto3_session):
            clean_up_spot(stack_name, boto3_session, disable_tqdm=True)
        else:
            terminate_running_agents(
                stack_output[constants.cloudformation.AGENT_TAG_NAME], boto3_session
            )

    try:
        if parameters:
            cfn.update_stack(
                StackName=stack_name,
                TemplateBody=template_body,
                Parameters=parameters,
                Capabilities=["CAPABILITY_IAM"],
                Tags=[
                    {
                        "Key": constants.deployment_types.TYPE_TAG_KEY,
                        "Value": deployment_type,
                    },
                ],
            )
        else:
            cfn.update_stack(
                StackName=stack_name,
                TemplateBody=template_body,
                Capabilities=["CAPABILITY_IAM"],
                Tags=[
                    {
                        "Key": constants.deployment_types.TYPE_TAG_KEY,
                        "Value": deployment_type,
                    },
                ],
            )
    except ClientError as e:
        if e.response["Error"]["Message"] != "No updates are to be performed.":
            raise e

        print(e.response["Error"]["Message"])

        ec2.start_instances(InstanceIds=[stack_output[constants.cloudformation.MASTER_ID]])
        start_waiter = ec2.get_waiter("instance_running")
        start_waiter.wait(
            InstanceIds=[stack_output[constants.cloudformation.MASTER_ID]],
            WaiterConfig={"Delay": 10},
        )
        return

    update_waiter.wait(StackName=stack_name, WaiterConfig={"Delay": 10})


def create_stack(
    stack_name: str,
    template_body: str,
    boto3_session: boto3.session.Session,
    deployment_type: str,
    parameters: Optional[List] = None,
) -> None:
    print(
        f"Creating stack {stack_name}. This may take a few minutes... "
        f"Check the CloudFormation Console for updates"
    )
    cfn = boto3_session.client("cloudformation")
    create_waiter = cfn.get_waiter("stack_create_complete")

    if parameters:
        cfn.create_stack(
            StackName=stack_name,
            TemplateBody=template_body,
            Parameters=parameters,
            Capabilities=["CAPABILITY_IAM"],
            Tags=[
                {
                    "Key": constants.defaults.STACK_TAG_KEY,
                    "Value": constants.defaults.STACK_TAG_VALUE,
                },
                {
                    "Key": constants.deployment_types.TYPE_TAG_KEY,
                    "Value": deployment_type,
                },
            ],
        )
    else:
        cfn.create_stack(
            StackName=stack_name,
            TemplateBody=template_body,
            Capabilities=["CAPABILITY_IAM"],
            Tags=[
                {
                    "Key": constants.defaults.STACK_TAG_KEY,
                    "Value": constants.defaults.STACK_TAG_VALUE,
                },
                {
                    "Key": constants.deployment_types.TYPE_TAG_KEY,
                    "Value": deployment_type,
                },
            ],
        )

    create_waiter.wait(StackName=stack_name, WaiterConfig={"Delay": 10})


def list_stacks(boto3_session: boto3.session.Session) -> List[Dict[str, Any]]:
    cfn = boto3_session.client("cloudformation")
    response = cfn.describe_stacks()

    output = []
    for stack in response["Stacks"]:
        for tag in stack["Tags"]:
            if (
                tag["Key"] == constants.defaults.STACK_TAG_KEY
                and tag["Value"] == constants.defaults.STACK_TAG_VALUE
            ):
                output.append(stack)
    return output


def get_output(stack_name: str, boto3_session: boto3.session.Session) -> Dict[str, str]:
    cfn = boto3_session.client("cloudformation")
    response = cfn.describe_stacks(StackName=stack_name)
    response_dict = {}

    try:
        stack_outputs = response["Stacks"][0]["Outputs"]
    except (KeyError, IndexError):
        raise NoStackOutputError(
            f"Stack {stack_name} is in an inconsistent state. "
            "Manual cleanup from the CloudFormation console may be needed."
        )

    for output in stack_outputs:
        k, v = output["OutputKey"], output["OutputValue"]
        response_dict[k] = v
    return response_dict


def get_params(stack_name: str, boto3_session: boto3.session.Session) -> Dict[str, str]:
    cfn = boto3_session.client("cloudformation")
    response = cfn.describe_stacks(StackName=stack_name)
    response_dict = {}
    params = response["Stacks"][0]["Parameters"]
    for param_obj in params:
        k = param_obj["ParameterKey"]
        v = param_obj["ParameterValue"]
        response_dict[k] = v
    return response_dict


def stack_uses_spot(stack_name: str, boto3_session: boto3.session.Session) -> bool:
    params = get_params(stack_name, boto3_session)
    if constants.cloudformation.SPOT_ENABLED not in params.keys():
        return False

    spot_enabled_str_val = params[constants.cloudformation.SPOT_ENABLED]
    if spot_enabled_str_val.lower() == "true":
        return True
    else:
        return False


def get_management_tag_key_value(stack_name: str) -> Tuple[str, str]:
    tag_key = f"det-{stack_name}"
    tag_val = f"det-agent-{stack_name}"
    return tag_key, tag_val


def deploy_stack(
    stack_name: str,
    template_body: str,
    keypair: str,
    boto3_session: boto3.session.Session,
    no_prompt: bool,
    deployment_type: str,
    parameters: Optional[List] = None,
    update_terminate_agents: bool = True,
) -> None:
    cfn = boto3_session.client("cloudformation")
    cfn.validate_template(TemplateBody=template_body)

    check_keypair(keypair, boto3_session)
    if stack_exists(stack_name, boto3_session):
        print("True - Updating Stack")

        if not no_prompt:
            tags = get_tags(stack_name, boto3_session)
            prompt_needed = False
            if constants.deployment_types.TYPE_TAG_KEY not in tags:
                print()
                print("Previous value of --deployment-type is unknown. Versions of `det` prior to")
                print("0.17.3 did not annotate deployed clusters, and it was the responsibility of")
                print("the user to make updates with the same --deployment-type. Note that if you")
                print("are sure --deployment-type was not set before, your cluster would have")
                print("deployed as --deployment-type simple (the default).")
                print()

                prompt_needed = True
            elif tags[constants.deployment_types.TYPE_TAG_KEY] != deployment_type:
                print("Value of --deployment-type has changed!")
                prompt_needed = True

            if prompt_needed:
                val = input(
                    "If --deployment-type has changed, updating the stack may erase the database.\n"
                    "Are you sure you want to proceed? [y/N]"
                )
                if val.lower() != "y":
                    print("Update canceled.")
                    sys.exit(1)

        update_stack(
            stack_name,
            template_body,
            boto3_session,
            deployment_type,
            parameters,
            update_terminate_agents=update_terminate_agents,
        )
    else:
        print("False - Creating Stack")

        create_stack(stack_name, template_body, boto3_session, deployment_type, parameters)


# EC2
def get_ec2_info(instance_id: str, boto3_session: boto3.session.Session) -> Dict:
    ec2 = boto3_session.client("ec2")

    response = ec2.describe_instances(InstanceIds=[instance_id])
    info = response["Reservations"][0]["Instances"][0]
    assert isinstance(info, dict), f"expected a dict of instance info but got {info}"
    return info


def check_keypair(name: str, boto3_session: boto3.session.Session) -> bool:
    ec2 = boto3_session.client("ec2")

    print(f"Checking if the SSH Keypair ({name}) exists:", end=" ")
    all_keys = ec2.describe_key_pairs()["KeyPairs"]
    names = [x["KeyName"] for x in all_keys]

    if name in names:
        print("True")
        return True

    print("False")
    print(
        f"Key pair {name} not found in {boto3_session.region_name}. "
        f"Please create the key pair {name} in {boto3_session.region_name} first"
    )
    sys.exit(1)


def terminate_running_agents(agent_tag_name: str, boto3_session: boto3.session.Session) -> None:
    ec2 = boto3_session.client("ec2")
    waiter = ec2.get_waiter("instance_terminated")

    response = ec2.describe_instances(
        Filters=[
            {"Name": "tag:Name", "Values": [agent_tag_name]},
            {"Name": "instance-state-name", "Values": ["running", "pending"]},
        ]
    )

    reservations = response["Reservations"]

    instance_ids = []
    for reservation in reservations:
        for instance in reservation["Instances"]:
            instance_ids.append(instance["InstanceId"])

    if instance_ids:
        ec2.terminate_instances(InstanceIds=instance_ids)
        for n in range(NUM_WAITS):
            print("Waiting For Agents To Terminate")
            try:
                waiter.wait(InstanceIds=instance_ids, WaiterConfig={"Delay": 10})
                break
            except WaiterError as e:
                if n == NUM_WAITS - 1:
                    raise e


# EC2 Spot
def list_spot_requests_for_stack(
    stack_name: str, boto3_session: boto3.session.Session
) -> List[Dict]:
    tag_key, tag_val = get_management_tag_key_value(stack_name)
    ec2 = boto3_session.client("ec2")
    response = ec2.describe_spot_instance_requests(
        Filters=[
            {"Name": f"tag:{tag_key}", "Values": [tag_val]},
            {"Name": "state", "Values": ["open", "active"]},
        ]
    )
    spot_requests = response["SpotInstanceRequests"]
    reqs = []
    for s in spot_requests:
        req = {
            "id": s["SpotInstanceRequestId"],
            "state": s["State"],
            "statusCode": s["Status"]["Code"],
            "statusMessage": s["Status"]["Message"],
            "instanceId": s.get("InstanceId", None),
        }
        reqs.append(req)
    return reqs


def delete_spot_requests_and_agents(
    stack_name: str, boto3_session: boto3.session.Session
) -> List[str]:
    """
    List all spot requests. Any requests that have an associated instance,
    terminate the instances (this will automatically cancel the spot
    request). Any requests that do not have an associated instance, cancel
    the spot requests.

    Returns the list of instance_ids that were deleted so at the end of spot
    cleanup, we can wait until all instances have been terminated.
    """
    spot_reqs = list_spot_requests_for_stack(stack_name, boto3_session)
    instances_to_del = []
    requests_to_term = []
    for req in spot_reqs:
        if req["instanceId"] is not None:
            instances_to_del.append(req["instanceId"])
        else:
            requests_to_term.append(req["id"])

    ec2 = boto3_session.client("ec2")

    if len(instances_to_del) > 0:
        ec2.terminate_instances(InstanceIds=instances_to_del)

    if len(requests_to_term) > 0:
        ec2.cancel_spot_instance_requests(SpotInstanceRequestIds=requests_to_term)

    return instances_to_del


def clean_up_spot(
    stack_name: str, boto3_session: boto3.session.Session, disable_tqdm: bool = False
) -> None:

    # The spot API is eventually consistent and the only way to guarantee
    # that we don't leave any spot requests alive (that may eventually be
    # fulfilled and lead to running EC2 instances) is to wait a long enough
    # period that any created spot requests will have shown up in the API.
    # 60 seconds seems like a relatively safe amount of time.
    SPOT_WAIT_SECONDS = 60

    start_time = time.time()

    all_terminated_instance_ids = set()

    format_str = "{l_bar}{bar}| (remaining time: {remaining})"
    pbar = tqdm.tqdm(
        total=SPOT_WAIT_SECONDS,
        desc="Cleaning up spot instances and spot instance requests",
        bar_format=format_str,
        disable=disable_tqdm,
    )
    progress_bar_state = 0.0
    while True:
        elapsed_time = time.time() - start_time
        if elapsed_time >= SPOT_WAIT_SECONDS:
            pbar.update(SPOT_WAIT_SECONDS - progress_bar_state)  # Exit TQDM with it showing 100%
            pbar.close()
            break

        tqdm_update = elapsed_time - progress_bar_state
        pbar.update(tqdm_update)
        progress_bar_state = elapsed_time

        instance_ids = delete_spot_requests_and_agents(stack_name, boto3_session)
        for i in instance_ids:
            all_terminated_instance_ids.add(i)

    # Final cleanup
    instance_ids = delete_spot_requests_and_agents(stack_name, boto3_session)
    for i in instance_ids:
        all_terminated_instance_ids.add(i)

    if len(instance_ids) > 0:
        ec2 = boto3_session.client("ec2")
        waiter = ec2.get_waiter("instance_terminated")
        for n in range(NUM_WAITS):
            print("Waiting For Spot Agents To Terminate")
            try:
                waiter.wait(InstanceIds=instance_ids, WaiterConfig={"Delay": 10})
                break
            except WaiterError as e:
                if n == NUM_WAITS - 1:
                    raise e


# S3
def empty_bucket(bucket_name: str, boto3_session: boto3.session.Session) -> None:
    s3 = boto3_session.resource("s3")
    try:
        bucket = s3.Bucket(bucket_name)
        bucket.objects.all().delete()

    except ClientError as e:
        if e.response["Error"]["Code"] != "NoSuchBucket":
            raise e
