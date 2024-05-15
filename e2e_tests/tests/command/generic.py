# tests covering generic ntscs

import pytest

from tests import api_utils
from tests import command as cmd

# created_id = api_utils.launch_ntsc(creds[0], workspaces[0].id, typ, experiment_id).id

def expect_logs()

@pytest.mark.e2e_cpu
def test_basic_ntcs_cli_messaging() -> None:
    sess = api_utils.user_session()
    with api_utils.test_workspace() as ws:
        lines = []  # type: List[str]
        with cmd.interactive_command(sess, ["notebook", "start"]) as notebook:
            for line in notebook.stdout:
                if re.search("", line) is not None:
                    return
                lines.append(line)

        raise ValueError(lines)



