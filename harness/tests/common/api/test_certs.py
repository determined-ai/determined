import pathlib
import shutil

from determined.common.api import certs
from tests import confdir

MOCK_MASTER_URL = "http://localhost:8080"
UNTRUSTED_CERT_PATH = pathlib.Path(__file__).parents[1] / "untrusted-root" / "127.0.0.1-ca.crt"


def test_cert_v0_upgrade() -> None:
    with confdir.use_test_config_dir() as config_dir:
        cert_path = config_dir / "master.crt"
        shutil.copy2(UNTRUSTED_CERT_PATH, cert_path)
        with cert_path.open() as fin:
            cert_data = fin.read()

        cert = certs.default_load(MOCK_MASTER_URL)
        assert isinstance(cert.bundle, str)
        with open(cert.bundle) as fin:
            loaded_cert_data = fin.read()
        assert loaded_cert_data.endswith(cert_data)
        assert not cert_path.exists()

        v1_certs_path = config_dir / "certs.json"
        assert v1_certs_path.exists()

        # Load once again from v1.
        cert2 = certs.default_load(MOCK_MASTER_URL)
        assert isinstance(cert2.bundle, str)
        with open(cert2.bundle) as fin:
            loaded_cert_data = fin.read()
        assert loaded_cert_data.endswith(cert_data)
