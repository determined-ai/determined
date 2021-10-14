import argparse
import sys

from determined import ipc

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("addr")
    parser.add_argument("cmd")
    parser.add_argument("cmd_args", nargs="*")
    args = parser.parse_args()

    addr = ipc.read_pid_server_addr(args.addr)

    with ipc.PIDClient(addr) as pid_client:
        sys.exit(pid_client.run_subprocess([args.cmd] + args.cmd_args))
