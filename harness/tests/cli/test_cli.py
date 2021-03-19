import os
import tempfile
from pathlib import Path

import pytest
import requests
import requests_mock

import determined.cli.cli as cli
import determined.cli.command as command
from determined.common import constants, context
from tests.filetree import FileTree

MINIMAL_CONFIG = '{"description": "test"}'


def test_parse_config() -> None:
    assert command.parse_config(None, [], [], []) == {}

    config = ["resources.slots=4"]
    assert command.parse_config(None, ["python", "train.py"], config, []) == {
        "resources": {"slots": 4},
        "entrypoint": ["python", "train.py"],
    }

    config = [
        "resources.slots=4",
    ]
    assert command.parse_config(None, ["python", "train.py"], config, []) == {
        "resources": {"slots": 4},
        "entrypoint": ["python", "train.py"],
    }

    config = ["""bind_mounts=host_path: /bin\ncontainer_path: /foo-bar"""]
    assert command.parse_config(None, [], config, ["/bin:/foo-bar2"]) == {
        "bind_mounts": [
            {"host_path": "/bin", "container_path": "/foo-bar"},
            {"host_path": "/bin", "container_path": "/foo-bar2"},
        ]
    }


def test_create_with_model_def(requests_mock: requests_mock.Mocker, tmp_path: Path) -> None:
    requests_mock.get("/info", status_code=200, json={"version": "1.0"})

    requests_mock.get(
        "/users/me", status_code=200, json={"username": constants.DEFAULT_DETERMINED_USER}
    )

    requests_mock.post("/login", status_code=200, json={"token": "fake-token"})

    requests_mock.post(
        "/experiments", status_code=requests.codes.created, headers={"Location": "/experiments/1"}
    )

    tempfile.mkstemp(dir=str(tmp_path))
    tempfile.mkstemp(dir=str(tmp_path))
    tempfile.mkstemp(dir=str(tmp_path))

    with FileTree(tmp_path, {"config.yaml": MINIMAL_CONFIG}) as tree:
        cli.main(
            ["experiment", "create", "--paused", str(tree.joinpath("config.yaml")), str(tmp_path)]
        )


@pytest.mark.slow  # type: ignore
def test_create_reject_large_model_def(requests_mock: requests_mock.Mocker, tmp_path: Path) -> None:
    requests_mock.get("/info", status_code=200, json={"version": "1.0"})

    requests_mock.get(
        "/users/me", status_code=200, json={"username": constants.DEFAULT_DETERMINED_USER}
    )

    requests_mock.post(
        "/experiments", status_code=requests.codes.created, headers={"Location": "/experiments/1"}
    )

    with tempfile.NamedTemporaryFile() as model_def_file:
        model_def_file.write(os.urandom(constants.MAX_CONTEXT_SIZE + 1))
        with FileTree(tmp_path, {"config.yaml": MINIMAL_CONFIG}) as tree, pytest.raises(SystemExit):
            cli.main(
                ["experiment", "create", str(tree.joinpath("config.yaml")), model_def_file.name]
            )


def test_read_context(tmp_path: Path) -> None:
    with FileTree(tmp_path, {"A.py": "", "B.py": "", "C.py": ""}) as tree:
        model_def, _ = context.read_context(tree)
        assert {f["path"] for f in model_def} == {"A.py", "B.py", "C.py"}


def test_read_context_with_detignore(tmp_path: Path) -> None:
    with FileTree(tmp_path, {"A.py": "", "B.py": "", "C.py": ""}) as tree:
        model_def, _ = context.read_context(tree)
        assert {f["path"] for f in model_def} == {"A.py", "B.py", "C.py"}

    with FileTree(tmp_path, {"A.py": "", "B.py": "", "C.py": "", ".detignore": "\nA.py\n"}) as tree:
        model_def, size = context.read_context(tree)
        assert {f["path"] for f in model_def} == {"B.py", "C.py"}

    with FileTree(tmp_path, {"A.py": "", "B.py": "", "C.py": "", ".detignore": "\n*.py\n"}) as tree:
        model_def, size = context.read_context(tree)
        assert model_def == []


def test_read_context_with_detignore_subdirs(tmp_path: Path) -> None:
    with FileTree(
        tmp_path,
        {
            "A.py": "",
            "B.py": "",
            Path("subdir").joinpath("A.py"): "",
            Path("subdir").joinpath("B.py"): "",
        },
    ) as tree:
        model_def, _ = context.read_context(tree)
        assert {f["path"] for f in model_def} == {
            "A.py",
            "B.py",
            "subdir",
            "subdir/A.py",
            "subdir/B.py",
        }

    with FileTree(
        tmp_path,
        {
            "A.py": "",
            "B.py": "",
            ".detignore": "\nA.py\n",
            Path("subdir").joinpath("A.py"): "",
            Path("subdir").joinpath("B.py"): "",
        },
    ) as tree:
        model_def, size = context.read_context(tree)
        assert {f["path"] for f in model_def} == {"B.py", "subdir", "subdir/B.py"}

    with FileTree(
        tmp_path,
        {
            "A.py": "",
            "B.py": "",
            Path("subdir").joinpath("A.py"): "",
            Path("subdir").joinpath("B.py"): "",
            ".detignore": "\nsubdir/A.py\n",
        },
    ) as tree:
        model_def, size = context.read_context(tree)
        assert {f["path"] for f in model_def} == {"A.py", "B.py", "subdir", "subdir/B.py"}

    with FileTree(
        tmp_path,
        {
            "A.py": "",
            "B.py": "",
            Path("subdir").joinpath("A.py"): "",
            Path("subdir").joinpath("B.py"): "",
            ".detignore": "\n*.py\n",
        },
    ) as tree:
        model_def, size = context.read_context(tree)
        assert len(model_def) == 1

    with FileTree(
        tmp_path,
        {"A.py": "", "B.py": "", "subdir/A.py": "", "subdir/B.py": "", ".detignore": "\nsubdir\n"},
    ) as tree:
        model_def, size = context.read_context(tree)
        assert {f["path"] for f in model_def} == {"A.py", "B.py"}

    with FileTree(
        tmp_path,
        {
            "A.py": "",
            "B.py": "",
            "subdir/A.py": "",
            "subdir/B.py": "",
            ".detignore": "\nsubdir/\n",
        },
    ) as tree:
        model_def, size = context.read_context(tree)
        assert {f["path"] for f in model_def} == {"A.py", "B.py"}


def test_read_context_ignore_pycaches(tmp_path: Path) -> None:
    with FileTree(
        tmp_path,
        {
            "__pycache__/A.cpython-37.pyc": "",
            "A.py": "",
            "subdir/A.py": "",
            "subdir/__pycache__/A.cpython-37.pyc": "",
        },
    ) as tree:
        model_def, _ = context.read_context(tree)
        assert {f["path"] for f in model_def} == {"A.py", "subdir", "subdir/A.py"}
