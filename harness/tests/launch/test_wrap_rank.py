import subprocess
import sys
import textwrap

from determined.launch import wrap_rank


def test_split_on_new_lines_or_carriage_returns() -> None:
    script = textwrap.dedent(
        r"""
        print("line with lf", end="\n", flush=True)
        input()
        print("line with cr", end="\r", flush=True)
        input()
        """
    )
    cmd = [
        sys.executable,
        "-u",
        wrap_rank.__file__,
        "--no-redirect-stdio",
        "0",
        sys.executable,
        "-u",
        "-c",
        script,
    ]
    p = subprocess.Popen(
        cmd,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    # Just for mypy.
    assert p.stdin and p.stdout and p.stderr
    assert p.stdout.readline() == b"[rank=0] line with lf\n"
    p.stdin.write(b"\n")
    p.stdin.flush()
    assert p.stdout.readline() == b"[rank=0] line with cr\n"
    p.stdin.write(b"\n")
    p.stdin.flush()
    assert p.wait() == 0
