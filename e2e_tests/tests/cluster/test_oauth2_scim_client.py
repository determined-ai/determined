import json
import subprocess

import pytest

from determined.errors import EnterpriseOnlyError
from determined.experimental import Determined
from tests import config
from tests import config as conf
from tests.cluster.test_users import ADMIN_CREDENTIALS, log_in_user


@pytest.fixture()
def is_ee() -> bool:
    command = [
        "det",
        "-m",
        config.make_master_url(),
        "master",
        "info",
        "--json",
    ]

    log_in_user(ADMIN_CREDENTIALS)
    output = subprocess.check_output(command, universal_newlines=True, stderr=subprocess.PIPE)

    rp = json.loads(output)["branding"]
    return rp == "hpe"


@pytest.mark.e2e_cpu
def test_list_oauth_clients(is_ee: bool) -> None:
    log_in_user(ADMIN_CREDENTIALS)
    det_obj = Determined(master=conf.make_master_url())
    user = det_obj.whoami()
    print(user.username)

    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "oauth",
        "client",
        "list",
    ]

    if is_ee:
        det_obj.list_oauth_clients()
        subprocess.run(command, check=True)
    else:
        with pytest.raises(EnterpriseOnlyError):
            det_obj.list_oauth_clients()
        with pytest.raises(subprocess.CalledProcessError):
             subprocess.run(command, check=True)


@pytest.mark.e2e_cpu
def test_add_client(is_ee: bool) -> None:
    log_in_user(ADMIN_CREDENTIALS)

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

    if is_ee:
        client = det_obj.add_oauth_client(domain="XXX", name="sdk_oauth_client_test")
        assert client.name == "sdk_oauth_client_test"
        assert client.domain == "XXX"
        subprocess.run(command, check=True)
    else:
        with pytest.raises(EnterpriseOnlyError):
            det_obj.add_oauth_client(domain="XXX", name="sdk_oauth_client_test")
        with pytest.raises(subprocess.CalledProcessError):
             subprocess.run(command, check=True)


@pytest.mark.e2e_cpu
def test_remove_client(is_ee: bool) -> None:
    log_in_user(ADMIN_CREDENTIALS)
    det_obj = Determined(master=conf.make_master_url())
    if is_ee:
        client = det_obj.add_oauth_client(domain="XXX", name="sdk_oauth_client_test")
        remove_id = client.id
        det_obj.remove_oauth_client(client_id=remove_id)
        list_client_ids = [oclient.id for oclient in det_obj.list_oauth_clients()]
        assert remove_id not in list_client_ids

        client = det_obj.add_oauth_client(domain="XXX", name="cli_oauth_client_test")
        remove_id = client.id

        command = [
            "det",
            "-m",
            conf.make_master_url(),
            "oauth",
            "client",
            "remove",
            str(remove_id),
        ]
        subprocess.run(command, check=True)
        list_client_ids = [oclient.id for oclient in det_obj.list_oauth_clients()]
        assert remove_id not in list_client_ids

    else:
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
