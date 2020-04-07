#!/usr/bin/env python

import argparse

# The hash function used here uses a random seed everytime the python interpreter is spawned and
# thus reporting different hashes for the same input. To fix the seed provide it through
# PYTHONHASHSEED environment variable.

# Typical Ephemeral port range on linux https://en.wikipedia.org/wiki/Ephemeral_port.
# The effective range is accessible via the /proc file system at node
# /proc/sys/net/ipv4/ip_local_port_range.

MIN_PORT = 5000
MAX_PORT = 5255
MAX_NUM_PORTS = 8


def run_num_to_port(run: int) -> int:
    start_port = MIN_PORT + (run * MAX_NUM_PORTS)
    if start_port + MAX_NUM_PORTS > MAX_PORT:
        raise OverflowError
    return start_port


def main() -> None:
    parser = argparse.ArgumentParser(description="Port helper.")
    parser.add_argument(
        "--run-number",
        help="get a port unique to this run of the determined cluster",
        type=int,
        required=True,
    )
    args = parser.parse_args()
    print(run_num_to_port(args.run_number))


if __name__ == "__main__":
    main()
