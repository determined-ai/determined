#!/bin/env python3
import os
import re
import sys
import subprocess

from typing import List, Generator, Callable
from datetime import datetime
from contextlib import contextmanager
from multiprocessing import Process


USAGE = """
Purpose:
    Launches a "det tensorboard start <args>" process and streams output to files.
    This solves the log buffer limit of 200 lines.
Usage:
    {script} <det tensorboard start args>

Example:
    {script} -t 40 41 42 -c /path/to/my/context/

Note:
    - Must not pass '-d' or '--detach' options
""".format(script=sys.argv[0])

# Note: there are some escape seq/control chars in output, strip these later.
# Scheduling TensorBoard (newly-desired-longhorn) (id: 40faa095-08ed-4c9b-888a-6a25f5c11efc)
TB_ID_REGEX = re.compile(r"Scheduling TensorBoard \([^\)]+\) \(id: ([a-z0-9-]{36}).*")
LOG_REPORT_TICK_BYTES = 1000


def sout(msg):
    print(f"SCRIPT OUT: {msg}")


def check_args() -> None:
    if len(sys.argv) < 2:
        print(f"{USAGE}")
        sys.exit(1)


def launch_det_stream(det_args: List[str]) -> Generator:
    process = subprocess.Popen(det_args, stdout=subprocess.PIPE)
    for line in iter(process.stdout.readline, b""):
        yield line.decode("utf-8")


def log_tee(gen: Generator, write_func: Callable) -> Generator:
    for line in gen:
        write_func(line)
        yield line


def log_report(gen: Generator, write_func: Callable) -> Generator:
    tick = 0
    bytes_written = 0
    for line in gen:
        bytes_written += write_func(line)

        cur_tick = bytes_written // LOG_REPORT_TICK_BYTES
        if tick < cur_tick:
            tick = cur_tick
            yield f"--> log_report(): bytes written: {bytes_written}"

    yield f"--> log_report(): stream ended: bytes written: {bytes_written}"


def det_logger_process(report_gen: Callable) -> None:
    for report in report_gen:
        print(f"{report}")


@contextmanager
def get_write(logpath: str) -> Callable:
    # open() doesn't have a way to fail if exists on call
    fp = None
    try:
        fp = os.open(logpath, os.O_WRONLY | os.O_EXCL | os.O_CREAT)

        def write_func(data: str) -> int:
            return os.write(fp, data.encode("utf-8"))

        yield write_func
    except FileExistsError as e:
        raise RuntimeError(f'Will not replace log file "{logpath}": {e}')
    finally:
        if fp is not None:
            os.close(fp)


def main(det_args: List[str]) -> None:
    date_str = datetime.now().strftime("%Y-%m-%dT%H%M%S")
    cmd_lp = f"./det_cmd.{date_str}.out"
    log_lp = f"./det_log.{date_str}.out"

    with get_write(cmd_lp) as cmd_write, get_write(log_lp) as log_write:

        tb_id = None
        log_process = None

        # Launch a tensorboard and tee the output to a log file
        det_tb_args = [*"det tensorboard start".split(" "), *det_args]
        det_out = launch_det_stream(det_tb_args)

        sout(f'Logging TensorBoard start output to: "{cmd_lp}"')
        for line in log_tee(det_out, cmd_write):
            sys.stdout.write(line)

            # Look for determined TensorBoard ID in output
            if tb_id is None:
                m = re.search(TB_ID_REGEX, line)
                if m is not None:
                    tb_id = m.group(1)
                    sout(f"Found TensorBoard ID: {tb_id}")

            # Launch logging stream from TensorBoard process
            if tb_id is not None and log_process is None:
                det_log_args = f"det tensorboard logs {tb_id} -f".split(" ")
                det_log_out = launch_det_stream(det_log_args)
                log_report_gen = log_report(det_log_out, log_write)

                sout("===> Launching TensorBoard logging process")
                sout(f"===> Logging to '{log_lp}'")
                log_process = Process(
                    target=det_logger_process,
                    args=(log_report_gen,),
                )
                log_process.start()

        sout("TensorBoard start command exited")
        if log_process is not None:
            sout("Joining on logging process")
            sout("===---> Press Ctrl+C to interrupt")
            log_process.join()


if __name__ == "__main__":
    check_args()
    main(sys.argv[1:])
