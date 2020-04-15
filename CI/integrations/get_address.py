import argparse

import boto3


def get_address_from_stack(stack_name: str) -> str:
    stack = boto3.resource("cloudformation").Stack(stack_name)
    instance_id = list(filter(lambda d: d["OutputKey"] == "MasterId", stack.outputs))[0][
        "OutputValue"
    ]
    private_ip = boto3.resource("ec2").Instance(instance_id).private_ip_address  # type: str
    return private_ip


def main() -> None:
    parser = argparse.ArgumentParser(description="Master address helper.")
    parser.add_argument(
        "stack_name", help="Name of CloudFormation stack to get master address from."
    )
    args = parser.parse_args()
    print(get_address_from_stack(args.stack_name), end="", flush=True)


if __name__ == "__main__":
    main()
