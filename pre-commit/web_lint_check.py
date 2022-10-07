import subprocess
import multiprocessing
import sys


def web_lint_check():
    if len(sys.argv) <= 2:
        print("At least 2 arguments are required")
        print("usage: python web_lint_check.py [js | css | misc] [<file_paths>]")
        exit(2)

    dir = "webui/react/"
    target: str = sys.argv[1]
    file_paths: str = " ".join(
        [file_path.replace(dir, "") for file_path in sys.argv[2:]]
    )
    nproc: int = multiprocessing.cpu_count()
    run_command: list[str] = [
        "make",
        f"-j{nproc}",
        "-C",
        dir,
        "prettier",
        f"PRE_ARGS=-- -c {file_paths}",
    ]

    # TODO: replace it with `match` if we support python v3.10
    if target == "js":
        run_command += ["eslint", f"ES_ARGS={file_paths}"]
    elif target == "css":
        run_command += ["stylelint", f"ST_ARGS={file_paths}"]
    elif target == "misc":
        run_command += ["check-package-lock"]
    else:
        print(f"{target} is not found")
        exit(2)

    returncode: int = subprocess.call(run_command)
    exit(returncode)


if __name__ == "__main__":
    web_lint_check()
