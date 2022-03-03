import argparse
import signal
import sys
from typing import Optional

from determined import ipc


def read_action(opt: str, val: str) -> Optional[signal.Signals]:
    if val.lower() == "wait":
        return None
    out = {s.name.lower(): s for s in signal.Signals}.get(val.lower())
    if out is None:
        raise ValueError(
            f"{opt} argument '{val}' is not valid; it should be a signal name ('SIGTERM', "
            "'SIGKILL', etc) or 'WAIT'"
        )
    return out


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("-x", "--on-fail", dest="on_fail", action="store", default="SIGTERM")
    parser.add_argument("-e", "--on-exit", dest="on_exit", action="store", default="WAIT")
    parser.add_argument("--grace-period", dest="grace_period", type=int, default=3)
    parser.add_argument("addr")
    parser.add_argument("num_workers", type=int)
    parser.add_argument("cmd")
    parser.add_argument("cmd_args", nargs="*")
    args = parser.parse_args()

    on_fail = read_action("--on-fail", args.on_fail)
    on_exit = read_action("--on-exit", args.on_exit)
    addr = ipc.read_pid_server_addr(args.addr)

    with ipc.PIDServer(addr, args.num_workers) as pid_server:
        sys.exit(
            pid_server.run_subprocess(
                cmd=[args.cmd] + args.cmd_args,
                on_fail=on_fail,
                on_exit=on_exit,
                grace_period=args.grace_period,
            ),
        )
