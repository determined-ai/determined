import numbers
import uuid
from typing import cast
from urllib import parse

import pytest
import requests

from tests.integrations import config as conf


def make_scim_url(path: str) -> str:
    return cast(str, parse.urljoin(conf.make_master_url(), "/scim/v2" + path))


def get_random_string() -> str:
    return str(uuid.uuid4())


@pytest.mark.e2e_cpu  # type: ignore
def test_create_scim_user() -> None:
    username = get_random_string()
    external_id = get_random_string()

    user_req = {
        "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
        "userName": username,
        "externalId": external_id,
        "active": True,
        "name": {"familyName": get_random_string(), "givenName": get_random_string()},
    }

    r = requests.post(
        make_scim_url("/Users"), auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD), json=user_req
    )
    r.raise_for_status()

    user_resp = r.json()

    assert user_resp.get("userName") == username
    assert user_resp.get("active")

    user_id = user_resp["id"]

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


@pytest.mark.e2e_cpu  # type: ignore
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


@pytest.mark.e2e_cpu  # type: ignore
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


@pytest.mark.e2e_cpu  # type: ignore
def test_okta_search_user_not_found() -> None:
    nonexistent_user = get_random_string()

    r = requests.get(
        make_scim_url("/Users"),
        auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD),
        params={"count": "100", "filter": f'userName eq "{nonexistent_user}"', "startIndex": "1"},
    )

    assert r.status_code == 200

    resp = r.json()

    assert "urn:ietf:params:scim:api:messages:2.0:ListResponse" in resp["schemas"]
    assert resp["totalResults"] == 0


@pytest.mark.e2e_cpu  # type: ignore
def test_okta_user_not_found() -> None:
    nonexistent_user = get_random_string()

    r = requests.get(
        make_scim_url(f"/Users/{nonexistent_user}"),
        auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD),
    )

    assert r.status_code == 404

    resp = r.json()

    assert len(resp["detail"]) > 0
    assert "urn:ietf:params:scim:api:messages:2.0:Error" in resp["schemas"]


@pytest.mark.e2e_cpu  # type: ignore
def test_okta_create_user() -> None:
    username = get_random_string() + "@okta.example.com"
    given_name = get_random_string()
    family_name = get_random_string()

    user_req = {
        "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
        "userName": username,
        "name": {"givenName": given_name, "familyName": family_name},
        "emails": [
            {"primary": True, "value": get_random_string() + "@okta.example.com", "type": "work"}
        ],
        "displayName": get_random_string(),
        "externalId": get_random_string(),
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

    assert resp["active"] is True
    assert len(user_id) > 0
    assert resp["name"]["familyName"] == family_name
    assert resp["name"]["givenName"] == given_name
    assert resp["userName"] == username
    assert "urn:ietf:params:scim:schemas:core:2.0:User" in resp["schemas"]

    r = requests.get(
        make_scim_url(f"/Users/{user_id}"),
        auth=(conf.SCIM_USERNAME, conf.SCIM_PASSWORD),
    )

    assert r.status_code == 200

    resp = r.json()

    assert resp["name"]["familyName"] == family_name
    assert resp["name"]["givenName"] == given_name
    assert resp["userName"] == username

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
