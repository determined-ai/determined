import argparse

import boto3


def get_output_from_stack(stack_name: str, output_key: str) -> str:
    stack = boto3.resource("cloudformation").Stack(stack_name)
    output_value = list(filter(lambda d: d["OutputKey"] == output_key, stack.outputs))[0][
        "OutputValue"
    ]  # type: str
    return output_value


def main() -> None:
    parser = argparse.ArgumentParser(description="Master address helper.")
    parser.add_argument(
        "stack_name", help="Name of CloudFormation stack to get master address from."
    )
    parser.add_argument("output_key", help="CloudFormation stack output key")
    args = parser.parse_args()
    print(get_output_from_stack(args.stack_name, args.output_key), end="", flush=True)


if __name__ == "__main__":
    main()
