import os
import tempfile
import uuid
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


def test_uuid_prefix(requests_mock: requests_mock.Mocker) -> None:
    # Create two UUIDs that are different at a known index.
    fake_uuid1 = str(uuid.uuid4())
    replace_ind = 4
    fake_uuid2 = (
        fake_uuid1[:replace_ind]
        + ("1" if fake_uuid1[replace_ind] == "0" else "0")
        + fake_uuid1[replace_ind + 1 :]
    )

    requests_mock.get("/info", status_code=200, json={"version": "1.0"})
    requests_mock.get(
        "/users/me", status_code=200, json={"username": constants.DEFAULT_DETERMINED_USER}
    )

    requests_mock.get(
        "/api/v1/shells",
        status_code=requests.codes.ok,
        json={"shells": [{"id": fake_uuid1}, {"id": fake_uuid2}]},
    )

    requests_mock.get(
        f"/api/v1/shells/{fake_uuid1}",
        status_code=requests.codes.ok,
        json={"config": None},
    )

    # Succeed with a full UUID.
    cli.main(["shell", "config", fake_uuid1])
    # Succeed with a partial unique prefix.
    cli.main(["shell", "config", fake_uuid1[: replace_ind + 1]])
    # Fail with an existing but nonunique prefix.
    with pytest.raises(SystemExit):
        cli.main(["shell", "config", fake_uuid1[:replace_ind]])
    # Fail with a nonexistent prefix.
    with pytest.raises(SystemExit):
        cli.main(["shell", "config", "x"])


@pytest.mark.slow
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
        model_def = context.read_legacy_context(tree)
        assert {f["path"] for f in model_def} == {"A.py", "B.py", "C.py"}


def test_read_context_with_detignore(tmp_path: Path) -> None:
    with FileTree(tmp_path, {"A.py": "", "B.py": "", "C.py": ""}) as tree:
        model_def = context.read_legacy_context(tree)
        assert {f["path"] for f in model_def} == {"A.py", "B.py", "C.py"}

    with FileTree(tmp_path, {"A.py": "", "B.py": "", "C.py": "", ".detignore": "\nA.py\n"}) as tree:
        model_def = context.read_legacy_context(tree)
        assert {f["path"] for f in model_def} == {"B.py", "C.py"}

    with FileTree(tmp_path, {"A.py": "", "B.py": "", "C.py": "", ".detignore": "\n*.py\n"}) as tree:
        model_def = context.read_legacy_context(tree)
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
        model_def = context.read_legacy_context(tree)
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
        model_def = context.read_legacy_context(tree)
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
        model_def = context.read_legacy_context(tree)
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
        model_def = context.read_legacy_context(tree)
        assert len(model_def) == 1

    with FileTree(
        tmp_path,
        {"A.py": "", "B.py": "", "subdir/A.py": "", "subdir/B.py": "", ".detignore": "\nsubdir\n"},
    ) as tree:
        model_def = context.read_legacy_context(tree)
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
        model_def = context.read_legacy_context(tree)
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
        model_def = context.read_legacy_context(tree)
        assert {f["path"] for f in model_def} == {"A.py", "subdir", "subdir/A.py"}


def test_cli_args_exist() -> None:
    valid_cmds = [
        "auth",
        "agent",
        "a",
        "command",
        "cmd",
        "checkpoint",
        "c",
        "deploy",
        "d",
        "experiment",
        "e",
        "master",
        "m",
        "model",
        "m",
        "notebook",
        "oauth",
        "resources",
        "res",
        "shell",
        "slot",
        "s",
        "task",
        "template",
        "tpl",
        "tensorboard",
        "trial",
        "t",
        "user",
        "u",
    ]
    for cmd in valid_cmds:
        cli.main([cmd, "help"])

    cli.main([])
    for cmd in ["version", "help"]:
        cli.main([cmd])

    with pytest.raises(SystemExit) as e:
        cli.main(["preview-search", "-h"])
    assert e.value.code == 0
