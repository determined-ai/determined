from determined.common.util import sizeof_fmt
from determined.util import _dict_to_list, _list_to_dict, sizeof_dict
import random
import string


def test_list_to_dict() -> None:
    r = _list_to_dict([{"a": 1}, {"b": 2}, {"a": 2}])
    assert r == {"a": [1, 2], "b": [2]}


def test_dict_to_list() -> None:
    r = _dict_to_list({"a": [1, 2], "b": [3, 4]})
    assert r == [{"a": 1, "b": 3}, {"a": 2, "b": 4}]


def test_sizeof_fmt() -> None:
    assert sizeof_fmt(1024) == "1.0KB"
    assert sizeof_fmt(36) == "36.0B"


def test_sizeof_dict() -> None:
    rand_values = [random.randint(0, 100) for i in range(10)]
    key = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
    dict = {
        key: rand_values
    }

    dict_size = len("{") + len("'") + len(str(key)) + len("'") + len(": ") + len(str(rand_values)) + len("}")
    assert sizeof_dict(dict) == dict_size