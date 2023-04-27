import argparse
from typing import Optional

import boto3


def get_output_from_stack(stack_name: str, output_key: str) -> Optional[str]:
    stack = boto3.resource("cloudformation").Stack(stack_name)
    if stack.outputs is None:
        return None
    outputs = list(filter(lambda d: d.get("OutputKey", None) == output_key, stack.outputs))
    if len(outputs) < 1:
        return None
    return outputs[0].get("OutputValue", None)


def main() -> None:
    parser = argparse.ArgumentParser(description="Master address helper.")
    parser.add_argument(
        "stack_name", help="Name of CloudFormation stack to get master address from."
    )
    parser.add_argument("output_key", help="CloudFormation stack output key")
    args = parser.parse_args()
    output = get_output_from_stack(args.stack_name, args.output_key)
    if output is None:
        raise RuntimeError(f"Could not find output {args.output_key} in stack {args.stack_name}")
    print(output, end="", flush=True)


if __name__ == "__main__":
    main()
