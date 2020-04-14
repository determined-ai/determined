from determined.util import _dict_to_list, _list_to_dict
from determined_common.util import sizeof_fmt


def test_list_to_dict() -> None:
    r = _list_to_dict([{"a": 1}, {"b": 2}, {"a": 2}])
    assert r == {"a": [1, 2], "b": [2]}


def test_dict_to_list() -> None:
    r = _dict_to_list({"a": [1, 2], "b": [3, 4]})
    assert r == [{"a": 1, "b": 3}, {"a": 2, "b": 4}]


def test_sizeof_fmt() -> None:
    assert sizeof_fmt(1024) == "1.0KB"
    assert sizeof_fmt(36) == "36.0B"
