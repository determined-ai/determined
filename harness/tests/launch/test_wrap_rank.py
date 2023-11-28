import subprocess
import sys
import textwrap

from determined.launch import wrap_rank


def test_split_on_new_lines_or_carriage_returns() -> None:
    script = textwrap.dedent(
        r"""
        import sys
        print("line with lf", end="\n", flush=True)
        print("check lf", file=sys.stderr, flush=True)
        input()
        print("line with cr", end="\r", flush=True)
        print("check cr", file=sys.stderr, flush=True)
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
    assert p.stdin and p.stdout and p.stderr
    # Wait for the first check on stderr.
    assert p.stderr.readline() == b"[rank=0] check lf\n"
    # Ensure we have the first line from stdout.
    assert p.stdout.readline() == b"[rank=0] line with lf\n"
    # Let the process proceed.
    p.stdin.write(b"\n")
    p.stdin.flush()
    # Again, for the carriage return line.
    assert p.stderr.readline() == b"[rank=0] check cr\n"
    assert p.stdout.readline() == b"[rank=0] line with cr\n"
    p.stdin.write(b"\n")
    p.stdin.flush()
    assert p.wait() == 0
