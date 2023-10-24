import pathlib
import shutil
import tempfile

from determined.common import util


def test_yaml_safe_load_strings() -> None:
    input_text = "asdf: 1\n"
    expect = {"asdf": 1}

    assert util.yaml_safe_load(input_text) == expect


def test_yaml_safe_load_files() -> None:
    input_text = "asdf: 1\n"
    expect = {"asdf": 1}

    d = pathlib.Path(tempfile.mkdtemp())
    try:
        path = d / "temp"
        path.write_text(input_text)
        with path.open() as f:
            assert util.yaml_safe_load(f) == expect
    finally:
        shutil.rmtree(d)


def test_yaml_safe_dump_strings() -> None:
    input_obj = {"asdf": 1}
    expect = "asdf: 1\n"

    assert util.yaml_safe_dump(input_obj, default_flow_style=False) == expect


def test_yaml_safe_dump_files() -> None:
    input_obj = {"asdf": 1}
    expect = "asdf: 1\n"

    d = pathlib.Path(tempfile.mkdtemp())
    try:
        path = d / "temp"
        with path.open("w") as f:
            util.yaml_safe_dump(input_obj, stream=f, default_flow_style=False)
        got_text = path.read_text()
        assert got_text == expect
    finally:
        shutil.rmtree(d)
