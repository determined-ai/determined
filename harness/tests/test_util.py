import os
import pathlib
from typing import Optional

import numpy as np
import pytest

import determined as det
from determined import util


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


@pytest.mark.parametrize("whats_there", [None, "dir", "file", "symlink"])
def test_force_create_symlink(whats_there: Optional[str], tmp_path: pathlib.Path) -> None:
    symlink_to_create = tmp_path.joinpath("tensorboard")
    symlink_source = tmp_path.joinpath("tensorboard-foo-0")

    os.makedirs(tmp_path.joinpath(symlink_source))

    if whats_there == "dir":
        os.makedirs(symlink_to_create)
    elif whats_there == "file":
        with symlink_to_create.open("w"):
            pass
    elif whats_there == "symlink":
        another_file = tmp_path.joinpath("another_file")
        with another_file.open("w"):
            pass
        os.symlink(another_file, symlink_to_create)

    util.force_create_symlink(str(symlink_source), str(symlink_to_create))

    expected_entry_found = False
    with os.scandir(tmp_path) as it:
        for entry in it:
            print(f"{entry}")
            if entry.name == "tensorboard":
                expected_entry_found = True
                assert entry.is_dir(follow_symlinks=True)
                assert entry.is_symlink()

    assert expected_entry_found
    assert os.readlink(str(symlink_to_create)) == str(symlink_source)

    if whats_there == "symlink":
        assert tmp_path.joinpath("another_file").exists(), "deleted previous symlink source"


def test_is_not_numerical_scalar() -> None:
    # Invalid types
    assert not util.is_numerical_scalar("foo")
    assert not util.is_numerical_scalar(np.array("foo"))
    assert not util.is_numerical_scalar(object())

    # Invalid shapes
    assert not util.is_numerical_scalar([1])
    assert not util.is_numerical_scalar(np.array([3.14]))
    assert not util.is_numerical_scalar(np.ones(shape=(5, 5)))


def test_is_numerical_scalar() -> None:
    assert util.is_numerical_scalar(1)
    assert util.is_numerical_scalar(1.0)
    assert util.is_numerical_scalar(-3.14)
    assert util.is_numerical_scalar(np.ones(shape=()))
    assert util.is_numerical_scalar(np.array(1))
    assert util.is_numerical_scalar(np.array(-3.14))
    assert util.is_numerical_scalar(np.array([1.0])[0])
