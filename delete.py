import boto3

from determined_deploy.aws.aws import delete


boto3_session = boto3.Session(region_name="us-west-2")
stacks = ["e2e-gpu-fdcfc1c-8600-0"]
for stack_name in stacks:
    delete(stack_name, boto3_session)
