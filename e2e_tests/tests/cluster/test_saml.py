import os

import pexpect
import pytest
from selenium import webdriver
from selenium.webdriver.common import by, keys
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support import ui

from determined.common.api import bindings
from tests import api_utils


@pytest.mark.e2e_saml
def test_saml_login() -> None:
    if "OKTA_CI_USER" not in os.environ:
        pytest.fail("OKTA_CI_PASS env var not set")
    if "OKTA_CI_PASS" not in os.environ:
        pytest.fail("OIDC_PASS env var not set")

    username = os.environ["OKTA_CI_USER"]
    password = os.environ["OKTA_CI_PASS"]

    sess = api_utils.admin_session()
    master_config = bindings.get_GetMasterConfig(sess).config
    assert "saml" in master_config

    driver = webdriver.Chrome()
    driver.get("http://127.0.0.1:8080/saml/initiate?relayState=cli")

    current_url = driver.current_url

    assert "2564556" in driver.title
    elem = driver.find_element(by.By.NAME, "username")
    elem.clear()
    elem.send_keys(username)

    elem = driver.find_element(by.By.NAME, "password")
    elem.clear()
    elem.send_keys(password)

    elem.send_keys(keys.Keys.RETURN)

    ui.WebDriverWait(driver, 15).until(EC.url_changes(current_url))
    returned_url = driver.current_url
    driver.close()

    child = pexpect.spawn("det", ["auth", "login", "--headless"])
    child.expect("localhost URL?", timeout=5)
    child.sendline(returned_url)
    output = child.read().decode()
    child.wait()
    child.close()

    assert output[-(len(username) + 3) : -3] == username
