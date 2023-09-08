import re

import pytest

import determined as det
from determined.common import api
from determined.experimental import client as _client
from tests import api_utils, detproc


@pytest.mark.e2e_cpu
@api_utils.skipif_ee()
def test_list_oauth_clients() -> None:
    sess = api_utils.admin_session()
    det_obj = _client.Determined._from_session(sess)
    with pytest.raises(det.errors.EnterpriseOnlyError):
        det_obj.list_oauth_clients()

    command = ["det", "oauth", "client", "list"]
    detproc.check_error(sess, command, "enterprise")


@pytest.mark.e2e_cpu
@api_utils.skipif_ee()
def test_add_client() -> None:
    sess = api_utils.admin_session()
    det_obj = _client.Determined._from_session(sess)
    with pytest.raises(det.errors.EnterpriseOnlyError):
        det_obj.add_oauth_client(domain="XXX", name="sdk_oauth_client_test")
    command = ["det", "oauth", "client", "add", "XXX", "cli_test_oauth_client"]
    detproc.check_error(sess, command, "enterprise")


@pytest.mark.e2e_cpu
@api_utils.skipif_ee()
def test_remove_client() -> None:
    sess = api_utils.admin_session()
    det_obj = _client.Determined._from_session(sess)
    with pytest.raises(det.errors.EnterpriseOnlyError):
        det_obj.remove_oauth_client(client_id="3")

    command = ["det", "oauth", "client", "remove", "4"]
    detproc.check_error(sess, command, "enterprise")


@pytest.mark.test_oauth
@api_utils.skipif_not_ee()
def test_list_oauth_clients_ee() -> None:
    sess = api_utils.admin_session()

    # Test SDK
    det_obj = _client.Determined._from_session(sess)
    det_obj.list_oauth_clients()

    # Test CLI
    command = [
        "det",
        "oauth",
        "client",
        "list",
    ]
    detproc.check_output(sess, command)

    # non-admin users are not allowed to call Oauth API.
    sess = api_utils.user_session()
    det_obj = _client.Determined._from_session(sess)
    with pytest.raises(api.errors.ForbiddenException):
        det_obj.list_oauth_clients()


@pytest.mark.test_oauth
@api_utils.skipif_not_ee()
def test_add_remove_client_ee() -> None:
    sess = api_utils.admin_session()

    # Test SDK
    det_obj = _client.Determined._from_session(sess)
    client = det_obj.add_oauth_client(domain="XXXSDK", name="sdk_oauth_client_test")
    remove_id = client.id
    det_obj.remove_oauth_client(client_id=remove_id)
    list_client_ids = [oclient.id for oclient in det_obj.list_oauth_clients()]
    assert remove_id not in list_client_ids

    # Test CLI.
    command = [
        "det",
        "oauth",
        "client",
        "add",
        "XXXCLI",
        "cli_test_oauth_client",
    ]
    output = detproc.check_output(sess, command).split("\\n")[0]
    assert "ID" in output
    r = "(.*)ID:(\\s*)(.*)"
    m = re.match(r, output)
    assert m is not None
    remove_id = m.group(3)
    command = [
        "det",
        "oauth",
        "client",
        "remove",
        str(remove_id),  # only one OAuth client is allowed.
    ]
    detproc.check_output(sess, command)
    list_client_ids = [oclient.id for oclient in det_obj.list_oauth_clients()]
    assert remove_id not in list_client_ids

    # non-admin users are not allowed to call Oauth API.
    sess = api_utils.user_session()
    det_obj = _client.Determined._from_session(sess)
    with pytest.raises(api.errors.ForbiddenException):
        det_obj.add_oauth_client(domain="XXXSDK", name="sdk_oauth_client_test")

    # non-admin users are not allowed to call Oauth API.
    with pytest.raises(api.errors.ForbiddenException):
        det_obj.remove_oauth_client(client_id="non-admin-call")
