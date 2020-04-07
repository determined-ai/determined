import argparse
import json


def convert_log_to_trace(log_file: str, trace_file: str):
    with open(trace_file, "wt") as output, open(log_file, "rt") as input:
        events = [json.loads(line) for line in input]
        json.dump({"traceEvents": events}, output)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Convert Determined profile logs to a chrome://tracing compatible file."
    )
    parser.add_argument("log_file", type=str, help="a path to log file to translate")
    parser.add_argument("out_file", type=str, help="a path to place the output file")
    args = parser.parse_args()
    convert_log_to_trace(args.log_file, args.out_file)
