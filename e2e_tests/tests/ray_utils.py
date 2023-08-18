import os
import pathlib
import subprocess
from typing import List, Optional


def ray_job_submit(
    exp_path: pathlib.Path,
    command: List[str],
    submit_args: Optional[List[str]] = None,
    port: int = 8265,
) -> None:
    env = os.environ.copy()
    env["RAY_ADDRESS"] = f"http://localhost:{port}"
    submit_args = submit_args or []
    subprocess.run(
        [
            "ray",
            "job",
            "submit",
        ]
        + submit_args
        + [
            "--working-dir",
            str(exp_path),
            "--",
        ]
        + command,
        check=True,
        env=env,
    )
