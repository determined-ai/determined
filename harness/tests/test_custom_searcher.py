import tempfile
from pathlib import Path

from tests.custom_search_mocks import MockMasterSearchRunner, SimulateMaster
from tests.search_methods import ASHASearchMethod, RandomSearchMethod


def test_run_random_searcher_exp_mock_master() -> None:
    max_trials = 5
    max_concurrent_trials = 2
    max_length = 500

    with tempfile.TemporaryDirectory() as searcher_dir:
        search_method = RandomSearchMethod(max_trials, max_concurrent_trials, max_length)
        mock_master_obj = SimulateMaster(metric=1.0)
        search_runner = MockMasterSearchRunner(search_method, mock_master_obj, Path(searcher_dir))
        search_runner.run(exp_config={}, context_dir="", includes=None)

    assert search_method.created_trials == 5
    assert search_method.pending_trials == 0
    assert search_method.closed_trials == 5
    assert len(search_runner.state.trials_created) == search_method.created_trials
    assert len(search_runner.state.trials_closed) == search_method.closed_trials


def test_run_asha_batches_exp_mock_master(tmp_path: Path) -> None:
    max_length = 3000
    max_trials = 16
    num_rungs = 3
    divisor = 4

    search_method = ASHASearchMethod(max_length, max_trials, num_rungs, divisor)
    mock_master_obj = SimulateMaster(metric=1.0)
    search_runner = MockMasterSearchRunner(search_method, mock_master_obj, tmp_path)
    search_runner.run(exp_config={}, context_dir="", includes=None)

    assert search_method.asha_search_state.pending_trials == 0
    assert search_method.asha_search_state.completed_trials == 16
    assert len(search_runner.state.trials_closed) == len(
        search_method.asha_search_state.closed_trials
    )
