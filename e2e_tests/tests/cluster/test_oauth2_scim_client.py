import re
import subprocess

import pytest

import determined as det
from determined.common import api
from determined.experimental import client as _client
from tests import api_utils
from tests import config as conf


@pytest.mark.e2e_cpu
@api_utils.skipif_ee()
def test_list_oauth_clients() -> None:
    api_utils.configure_token_store(conf.ADMIN_CREDENTIALS)
    det_obj = _client.Determined(master=conf.make_master_url())
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "oauth",
        "client",
        "list",
    ]

    with pytest.raises(det.errors.EnterpriseOnlyError):
        det_obj.list_oauth_clients()
    with pytest.raises(subprocess.CalledProcessError):
        subprocess.run(command, check=True)


@pytest.mark.e2e_cpu
@api_utils.skipif_ee()
def test_add_client() -> None:
    api_utils.configure_token_store(conf.ADMIN_CREDENTIALS)

    det_obj = _client.Determined(master=conf.make_master_url())
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

    with pytest.raises(det.errors.EnterpriseOnlyError):
        det_obj.add_oauth_client(domain="XXX", name="sdk_oauth_client_test")
    with pytest.raises(subprocess.CalledProcessError):
        subprocess.run(command, check=True)


@pytest.mark.e2e_cpu
@api_utils.skipif_ee()
def test_remove_client() -> None:
    api_utils.configure_token_store(conf.ADMIN_CREDENTIALS)
    det_obj = _client.Determined(master=conf.make_master_url())
    with pytest.raises(det.errors.EnterpriseOnlyError):
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


@pytest.mark.test_oauth
@api_utils.skipif_not_ee()
def test_list_oauth_clients_ee() -> None:
    api_utils.configure_token_store(conf.ADMIN_CREDENTIALS)

    # Test SDK
    det_obj = _client.Determined(master=conf.make_master_url())
    det_obj.list_oauth_clients()

    # Test CLI
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "oauth",
        "client",
        "list",
    ]
    subprocess.run(command, check=True)

    # non-admin users are not allowed to call Oauth API.
    new_creds = api_utils.create_test_user()
    api_utils.configure_token_store(new_creds)
    with pytest.raises(api.errors.ForbiddenException):
        det_obj = _client.Determined(master=conf.make_master_url())
        det_obj.list_oauth_clients()


@pytest.mark.test_oauth
@api_utils.skipif_not_ee()
def test_add_remove_client_ee() -> None:
    api_utils.configure_token_store(conf.ADMIN_CREDENTIALS)

    # Test SDK.
    det_obj = _client.Determined(master=conf.make_master_url())
    client = det_obj.add_oauth_client(domain="XXXSDK", name="sdk_oauth_client_test")
    remove_id = client.id
    det_obj.remove_oauth_client(client_id=remove_id)
    list_client_ids = [oclient.id for oclient in det_obj.list_oauth_clients()]
    assert remove_id not in list_client_ids

    # Test CLI.
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "oauth",
        "client",
        "add",
        "XXXCLI",
        "cli_test_oauth_client",
    ]
    output = str(subprocess.check_output(command)).split("\\n")[0]
    assert "ID" in output
    r = "(.*)ID:(\\s*)(.*)"
    m = re.match(r, output)
    assert m is not None
    remove_id = m.group(3)
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "oauth",
        "client",
        "remove",
        str(remove_id),  # only one OAuth client is allowed.
    ]
    subprocess.run(command, check=True)
    list_client_ids = [oclient.id for oclient in det_obj.list_oauth_clients()]
    assert remove_id not in list_client_ids

    # non-admin users are not allowed to call Oauth API.
    new_creds = api_utils.create_test_user()
    api_utils.configure_token_store(new_creds)
    det_obj = _client.Determined(master=conf.make_master_url())
    with pytest.raises(api.errors.ForbiddenException):
        client = det_obj.add_oauth_client(domain="XXXSDK", name="sdk_oauth_client_test")

    # non-admin users are not allowed to call Oauth API.
    with pytest.raises(api.errors.ForbiddenException):
        det_obj.remove_oauth_client(client_id="non-admin-call")
