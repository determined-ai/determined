import pytest
import time

import determined.cli.cli as cli

@pytest.mark.e2e_cpu
def test_experiment_cli() -> None:
    with pytest.raises(SystemExit) as e:
        cli.main(['e', 'list', '--limit', '2', '--offset', '0'])
    assert e.value.code == 0
    print(resp)
