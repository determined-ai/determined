import io
import tarfile
from pathlib import Path

import responses
from responses import matchers

from determined.common import api
from determined.common.api import authentication
from determined.experimental import client


def get_long_str(approx_len: int) -> str:
    block = "12345678223456783234567842345678\n"
    s = io.StringIO()

    i = 0
    while i < approx_len:
        s.write(block)
        i += len(block)
    return s.getvalue()


mock_content = {
    "emptyDir": "",
    "data.txt": "This is mock data.",
    "lib/big-data.txt": get_long_str(1024 * 64),
    "lib/math.py": "def triple(x):\n  return x * 3",
    "print.py": "print('hello')",
}


def setup_mock_checkpoint(directory: Path) -> None:
    for k, v in mock_content.items():
        fpath = directory / k
        if len(v) == 0:
            # This is a directory
            fpath.mkdir(parents=True, exist_ok=True)
        else:
            fpath.parent.mkdir(parents=True, exist_ok=True)
            with open(fpath, "w") as f:
                f.write(v)


def verify_test_checkpoint(directory: Path) -> None:
    for k, v in mock_content.items():
        fpath = directory / k
        if len(v) == 0:
            # This is a directory
            assert fpath.exists()
        else:
            with open(fpath) as f:
                assert f.read() == v


def get_response_raw_tgz(checkpoint_path: Path) -> bytes:
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w|gz") as tf:
        for k in mock_content:
            tf.add(checkpoint_path / k, arcname=k)

    return buf.getbuffer()


@responses.activate
def test_checkpoint_download_via_master(tmp_path: Path) -> None:
    uuid_tgz = "dummy-uuid-123-tgz"
    checkpoint_path = tmp_path / "mock-checkpoint"

    setup_mock_checkpoint(checkpoint_path)

    # Set up mocks
    responses.get(
        f"https://dummy-master.none:443/checkpoints/{uuid_tgz}",
        body=get_response_raw_tgz(checkpoint_path),
        stream=True,
        status=200,
        match=[matchers.header_matcher({"Accept": "application/gzip"})],
    )

    checkpoint_path = tmp_path / uuid_tgz
    utp = authentication.UsernameTokenPair("username", "token")
    client.Checkpoint._download_via_master(
        api.Session("https://dummy-master.none", utp, cert=None),
        uuid_tgz,
        checkpoint_path,
    )
    verify_test_checkpoint(checkpoint_path)
