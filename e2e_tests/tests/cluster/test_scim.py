import numbers
from urllib import parse

import pytest
import requests

from tests import api_utils
from tests import config as conf


def make_scim_url(path: str) -> str:
    return parse.urljoin(conf.make_master_url(), "/scim/v2" + path)


@pytest.mark.e2e_cpu
@api_utils.skipif_scim_not_enabled()
def test_create_scim_user() -> None:
    username = api_utils.get_random_string()
    external_id = api_utils.get_random_string()

    user_req = {
        "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
        "userName": username,
        "externalId": external_id,
        "active": True,
        "name": {
            "familyName": api_utils.get_random_string(),
            "givenName": api_utils.get_random_string(),
        },
    }

    r = requests.post(
        make_scim_url("/Users"), auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD), json=user_req
    )
    r.raise_for_status()

    user_resp = r.json()

    user_id = user_resp["id"]
    user_loc = make_scim_url(f"/Users/{user_id}")

    assert user_resp.get("userName") == username
    assert user_resp.get("active")
    assert user_resp["meta"]["location"] == user_loc

    patch_req = {
        "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
        "operations": [{"op": "replace", "value": {"active": False}}],
    }

    r = requests.patch(
        make_scim_url(f"/Users/{user_id}"),
        auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD),
        json=patch_req,
    )
    r.raise_for_status()

    patched_resp = r.json()

    assert not patched_resp.get("active")

    search_req = {"filter": f'userName eq "{username}"'}
    r = requests.get(
        make_scim_url("/Users"), auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD), params=search_req
    )
    r.raise_for_status()

    search_resp = r.json()

    assert search_resp["totalResults"] == 1
    assert search_resp["startIndex"] == 1
    assert len(search_resp["Resources"]) == 1
    assert search_resp["Resources"][0]["userName"] == username
    assert search_resp["Resources"][0]["meta"]["location"] == user_loc

    indiv_user = requests.get(
        make_scim_url(f"/Users/{user_id}"), auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD)
    )
    indiv_user.raise_for_status()
    indiv_user_resp = indiv_user.json()

    assert indiv_user_resp.get("id") == user_id
    assert indiv_user_resp["meta"]["location"] == user_loc


@pytest.mark.e2e_cpu
@api_utils.skipif_scim_not_enabled()
def test_okta_get_users() -> None:
    r = requests.get(
        make_scim_url("/Users"),
        auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD),
        params={"count": "2", "startIndex": "1"},
    )

    assert r.status_code == 200

    resp = r.json()

    assert resp["Resources"] is not None
    assert "urn:ietf:params:scim:api:messages:2.0:ListResponse" in resp["schemas"]
    assert isinstance(resp["itemsPerPage"], numbers.Number)
    assert isinstance(resp["startIndex"], numbers.Number)
    assert isinstance(resp["totalResults"], numbers.Number)


@pytest.mark.e2e_cpu
@api_utils.skipif_scim_not_enabled()
def test_okta_get_groups() -> None:
    r = requests.get(
        make_scim_url("/Groups"),
        auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD),
        params={"count": "100", "startIndex": "1"},
    )

    assert r.status_code == 200

    resp = r.json()

    assert resp["Resources"] is not None
    assert "urn:ietf:params:scim:api:messages:2.0:ListResponse" in resp["schemas"]
    assert isinstance(resp["itemsPerPage"], numbers.Number)
    assert isinstance(resp["startIndex"], numbers.Number)
    assert isinstance(resp["totalResults"], numbers.Number)


@pytest.mark.e2e_cpu
@api_utils.skipif_scim_not_enabled()
def test_okta_search_user_not_found() -> None:
    nonexistent_user = api_utils.get_random_string()

    r = requests.get(
        make_scim_url("/Users"),
        auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD),
        params={"count": "100", "filter": f'userName eq "{nonexistent_user}"', "startIndex": "1"},
    )

    assert r.status_code == 200

    resp = r.json()

    assert "urn:ietf:params:scim:api:messages:2.0:ListResponse" in resp["schemas"]
    assert resp["totalResults"] == 0


@pytest.mark.e2e_cpu
@api_utils.skipif_scim_not_enabled()
def test_okta_user_not_found() -> None:
    nonexistent_user = api_utils.get_random_string()

    r = requests.get(
        make_scim_url(f"/Users/{nonexistent_user}"),
        auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD),
    )

    assert r.status_code == 404

    resp = r.json()

    assert len(resp["detail"]) > 0
    assert "urn:ietf:params:scim:api:messages:2.0:Error" in resp["schemas"]


@pytest.mark.e2e_cpu
@api_utils.skipif_scim_not_enabled()
def test_okta_create_user() -> None:
    username = api_utils.get_random_string() + "@okta.example.com"
    given_name = api_utils.get_random_string()
    family_name = api_utils.get_random_string()

    user_req = {
        "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
        "userName": username,
        "name": {"givenName": given_name, "familyName": family_name},
        "emails": [
            {
                "primary": True,
                "value": api_utils.get_random_string() + "@okta.example.com",
                "type": "work",
            }
        ],
        "displayName": api_utils.get_random_string(),
        "externalId": api_utils.get_random_string(),
        "groups": [],
        "active": True,
    }

    r = requests.post(
        make_scim_url("/Users"),
        auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD),
        json=user_req,
    )

    assert r.status_code == 201

    resp = r.json()

    user_id = resp["id"]
    user_loc = make_scim_url(f"/Users/{user_id}")

    assert resp["active"] is True
    assert len(user_id) > 0
    assert resp["name"]["familyName"] == family_name
    assert resp["name"]["givenName"] == given_name
    assert resp["userName"] == username
    assert "urn:ietf:params:scim:schemas:core:2.0:User" in resp["schemas"]
    assert resp["meta"]["location"] == user_loc

    r = requests.get(
        make_scim_url(f"/Users/{user_id}"),
        auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD),
    )

    assert r.status_code == 200

    resp = r.json()

    assert resp["name"]["familyName"] == family_name
    assert resp["name"]["givenName"] == given_name
    assert resp["userName"] == username
    assert resp["meta"]["location"] == user_loc

    patch_req = {
        "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
        "Operations": [{"op": "replace", "value": {"active": False}}],
    }

    r = requests.patch(
        make_scim_url(f"/Users/{user_id}"),
        auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD),
        json=patch_req,
    )

    assert r.status_code == 200

    resp = r.json()

    assert resp["active"] is False
