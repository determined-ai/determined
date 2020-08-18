import sys
from typing import Dict, List, Optional

import boto3
from botocore.exceptions import ClientError, WaiterError

from determined_deploy.aws import constants

# Try waiting for stack to delete this many times. We break up the waiting so the delete job
# will not fail CI.
NUM_WAITS = 5


def get_user(boto3_session: boto3.session.Session) -> str:
    sts = boto3_session.client("sts")
    response = sts.get_caller_identity()
    return response["Arn"].split("/")[-1]


def stop_master(master_id: str, boto3_session: boto3.session.Session):
    ec2 = boto3_session.client("ec2")
    waiter = ec2.get_waiter("instance_stopped")
    ec2.stop_instances(InstanceIds=[master_id])
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

    # First, shut down the master so no new agents are started.
    stack_output = get_output(stack_name, boto3_session)
    master_id = stack_output[constants.cloudformation.MASTER_ID]
    describe_instance_response = ec2.describe_instances(
        Filters=[{"Name": "instance-id", "Values": [master_id]}],
    )

    if describe_instance_response["Reservations"]:
        print("Stopping Master Instance")
        stop_master(master_id, boto3_session)

    # Second, terminate the agents so nothing can write to the checkpoint bucket. We create agent
    # instances outside of cloudformation, so we have to manually terminate them.
    print("Terminating Running Agents")
    terminate_running_agents(stack_output[constants.cloudformation.AGENT_TAG_NAME], boto3_session)
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
    parameters: Optional[List] = None,
) -> None:
    cfn = boto3_session.client("cloudformation")
    ec2 = boto3_session.client("ec2")
    update_waiter = cfn.get_waiter("stack_update_complete")

    print(
        f"Updating stack {stack_name}. This may take a few minutes... "
        f"Check the CloudFormation Console for updates"
    )
    stack_output = get_output(stack_name, boto3_session)

    stop_master(stack_output[constants.cloudformation.MASTER_ID], boto3_session)
    terminate_running_agents(stack_output[constants.cloudformation.AGENT_TAG_NAME], boto3_session)

    try:
        if parameters:
            cfn.update_stack(
                StackName=stack_name,
                TemplateBody=template_body,
                Parameters=parameters,
                Capabilities=["CAPABILITY_IAM"],
            )
        else:
            cfn.update_stack(
                StackName=stack_name, TemplateBody=template_body, Capabilities=["CAPABILITY_IAM"]
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
        )
    else:
        cfn.create_stack(
            StackName=stack_name, TemplateBody=template_body, Capabilities=["CAPABILITY_IAM"]
        )

    create_waiter.wait(StackName=stack_name, WaiterConfig={"Delay": 10})


def get_output(stack_name: str, boto3_session: boto3.session.Session) -> Dict[str, str]:
    cfn = boto3_session.client("cloudformation")
    response = cfn.describe_stacks(StackName=stack_name)
    response_dict = {}

    for output in response["Stacks"][0]["Outputs"]:
        k, v = output["OutputKey"], output["OutputValue"]
        response_dict[k] = v
    return response_dict


def deploy_stack(
    stack_name: str,
    template_body: str,
    keypair: str,
    boto3_session: boto3.session.Session,
    parameters: Optional[List] = None,
) -> None:
    cfn = boto3_session.client("cloudformation")
    cfn.validate_template(TemplateBody=template_body)

    check_keypair(keypair, boto3_session)
    if stack_exists(stack_name, boto3_session):
        print("True - Updating Stack")

        update_stack(stack_name, template_body, boto3_session, parameters)
    else:
        print("False - Creating Stack")

        create_stack(stack_name, template_body, boto3_session, parameters)


# EC2
def get_ec2_info(instance_id: str, boto3_session: boto3.session.Session) -> Dict:
    ec2 = boto3_session.client("ec2")

    response = ec2.describe_instances(InstanceIds=[instance_id])
    return response["Reservations"][0]["Instances"][0]


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
            {"Name": "instance-state-name", "Values": ["running"]},
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


# S3
def empty_bucket(bucket_name: str, boto3_session: boto3.session.Session) -> None:
    s3 = boto3_session.resource("s3")
    try:
        bucket = s3.Bucket(bucket_name)
        bucket.objects.all().delete()

    except ClientError as e:
        if e.response["Error"]["Code"] != "NoSuchBucket":
            raise e
