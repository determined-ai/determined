import subprocess

import pytest

from determined.errors import EnterpriseOnlyError
from determined.experimental import Determined
from tests import api_utils
from tests import config as conf
from tests.cluster.test_users import ADMIN_CREDENTIALS


@pytest.mark.e2e_cpu
def test_list_oauth_clients() -> None:
    api_utils.configure_token_store(ADMIN_CREDENTIALS)
    det_obj = Determined(master=conf.make_master_url())
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "oauth",
        "client",
        "list",
    ]

    with pytest.raises(EnterpriseOnlyError):
        det_obj.list_oauth_clients()
    with pytest.raises(subprocess.CalledProcessError):
        subprocess.run(command, check=True)


@pytest.mark.e2e_cpu
def test_add_client() -> None:
    api_utils.configure_token_store(ADMIN_CREDENTIALS)

    det_obj = Determined(master=conf.make_master_url())
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "oauth",
        "client",
        "add",
        "XXX",
        "cli_test_oauth_client",
    ]

    with pytest.raises(EnterpriseOnlyError):
        det_obj.add_oauth_client(domain="XXX", name="sdk_oauth_client_test")
    with pytest.raises(subprocess.CalledProcessError):
        subprocess.run(command, check=True)


@pytest.mark.e2e_cpu
def test_remove_client() -> None:
    api_utils.configure_token_store(ADMIN_CREDENTIALS)
    det_obj = Determined(master=conf.make_master_url())
    with pytest.raises(EnterpriseOnlyError):
        det_obj.remove_oauth_client(client_id="3")
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "oauth",
        "client",
        "remove",
        "4",
    ]
    with pytest.raises(subprocess.CalledProcessError):
        subprocess.run(command, check=True)
