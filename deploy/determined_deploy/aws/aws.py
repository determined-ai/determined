import sys
from typing import Dict, List, Optional

import boto3
from botocore.exceptions import ClientError

from determined_deploy.aws import constants


def get_user(boto3_session: boto3.session.Session) -> str:
    sts = boto3_session.client("sts")
    response = sts.get_caller_identity()
    return response["Arn"].split("/")[-1]


def delete(stack_name: str, boto3_session: boto3.session.Session) -> None:
    ec2 = boto3_session.client("ec2")
    bucket_name = get_output(stack_name, boto3_session).get(
        constants.cloudformation.CHECKPOINT_BUCKET
    )
    if bucket_name:
        empty_bucket(bucket_name, boto3_session)

    stack_output = get_output(stack_name, boto3_session)
    master_id = stack_output[constants.cloudformation.MASTER_ID]
    describe_instance_response = ec2.describe_instances(
        Filters=[{"Name": "instance-id", "Values": [master_id]}],
    )

    if describe_instance_response["Reservations"]:
        ec2.stop_instances(InstanceIds=[master_id])
        ec2.modify_instance_attribute(
            Attribute="disableApiTermination", Value="false", InstanceId=master_id
        )

    terminate_running_agents(stack_output[constants.cloudformation.AGENT_TAG_NAME], boto3_session)
    delete_stack(stack_name, boto3_session)


# Cloudformation
def stack_exists(stack_name: str, boto3_session: boto3.session.Session) -> bool:
    cfn = boto3_session.client("cloudformation")

    try:
        cfn.describe_stacks(StackName=stack_name)
    except ClientError:
        print(f"{stack_name} not found")
        return False
    return True


def delete_stack(stack_name: str, boto3_session: boto3.session.Session) -> None:
    cfn = boto3_session.client("cloudformation")
    delete_waiter = cfn.get_waiter("stack_delete_complete")

    print(f"Deleting stack {stack_name}")
    cfn.delete_stack(StackName=stack_name)
    delete_waiter.wait(StackName=stack_name, WaiterConfig={"Delay": 10})


def update_stack(
    stack_name: str,
    template_body: str,
    boto3_session: boto3.session.Session,
    parameters: Optional[List] = None,
) -> None:
    print(f"Updating stack {stack_name}")
    cfn = boto3_session.client("cloudformation")
    ec2 = boto3_session.client("ec2")
    update_waiter = cfn.get_waiter("stack_update_complete")

    stack_output = get_output(stack_name, boto3_session)
    ec2.stop_instances(InstanceIds=[stack_output[constants.cloudformation.MASTER_ID]])

    stop_waiter = ec2.get_waiter("instance_stopped")
    stop_waiter.wait(
        InstanceIds=[stack_output[constants.cloudformation.MASTER_ID]], WaiterConfig={"Delay": 10},
    )

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
    print(f"Creating stack {stack_name}")
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
    boto3_session: boto3.session.Session,
    parameters: Optional[List] = None,
) -> None:
    cfn = boto3_session.client("cloudformation")
    cfn.validate_template(TemplateBody=template_body)

    if stack_exists(stack_name, boto3_session):
        update_stack(stack_name, template_body, boto3_session, parameters)
    else:
        create_stack(stack_name, template_body, boto3_session, parameters)


# EC2
def get_ec2_info(instance_id: str, boto3_session: boto3.session.Session) -> Dict:
    ec2 = boto3_session.client("ec2")

    response = ec2.describe_instances(InstanceIds=[instance_id])
    return response["Reservations"][0]["Instances"][0]


def check_keypair(name: str, boto3_session: boto3.session.Session) -> bool:
    ec2 = boto3_session.client("ec2")

    all_keys = ec2.describe_key_pairs()["KeyPairs"]
    names = [x["KeyName"] for x in all_keys]

    if name in names:
        return True

    print(f"Key pair {name} not found. Please create key pair first")
    sys.exit(1)


def terminate_running_agents(agent_tag_name: str, boto3_session: boto3.session.Session) -> None:
    ec2 = boto3_session.client("ec2")

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


# S3
def empty_bucket(bucket_name: str, boto3_session: boto3.session.Session) -> None:
    s3 = boto3_session.resource("s3")
    try:
        bucket = s3.Bucket(bucket_name)
        bucket.objects.all().delete()

    except ClientError as e:
        if e.response["Error"]["Code"] != "NoSuchBucket":
            raise e
