import pytest

import determined as det


def test_list_to_dict() -> None:
    r = det.util._list_to_dict([{"a": 1}, {"b": 2}, {"a": 2}])
    assert r == {"a": [1, 2], "b": [2]}


def test_dict_to_list() -> None:
    r = det.util._dict_to_list({"a": [1, 2], "b": [3, 4]})
    assert r == [{"a": 1, "b": 3}, {"a": 2, "b": 4}]


def test_sizeof_fmt() -> None:
    assert det.common.util.sizeof_fmt(1024) == "1.0KB"
    assert det.common.util.sizeof_fmt(36) == "36.0B"


def test_calculate_batch_sizes() -> None:
    # Valid cases.
    psbs, gbs = det.util.calculate_batch_sizes({"global_batch_size": 1}, 1, "Trial")
    assert (psbs, gbs) == (1, 1)
    psbs, gbs = det.util.calculate_batch_sizes({"global_batch_size": 8}, 2, "Trial")
    assert (psbs, gbs) == (4, 8)

    # Missing global_batch_size.
    with pytest.raises(det.errors.InvalidExperimentException, match="is a required hyperparameter"):
        det.util.calculate_batch_sizes({}, 1, "Trial")

    # Invalid global_batch_size.
    for x in ["1", 32.0]:
        with pytest.raises(det.errors.InvalidExperimentException, match="must be an integer value"):
            det.util.calculate_batch_sizes({"global_batch_size": x}, 1, "Trial")

    # global_batch_size too small.
    with pytest.raises(det.errors.InvalidExperimentException, match="to be greater or equal"):
        det.util.calculate_batch_sizes({"global_batch_size": 1}, 2, "Trial")
