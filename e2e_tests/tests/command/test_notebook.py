import re
from typing import List

import pytest

from tests import command as cmd


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_basic_notebook_start_and_kill() -> None:
    lines = []  # type: List[str]
    with cmd.interactive_command("notebook", "start") as notebook:
        for line in notebook.stdout:
            if re.search("Jupyter Notebook .*is running at", line) is not None:
                return
            lines.append(line)

    raise ValueError(lines)
