from typing import Optional, Tuple

import pytest

import tests.api_utils as api_utils
from determined.common import yaml
from determined.common.api import Session, bindings, errors
from tests import command as cmd
from tests import config as conf
from tests import template as tpl


@pytest.mark.e2e_cpu
def test_set_template() -> None:
    template_name = "test_set_template"
    template_path = conf.fixtures_path("templates/template.yaml")
    tpl.set_template(template_name, template_path)
    config = yaml.safe_load(tpl.describe_template(template_name))
    assert config == conf.load_config(template_path)


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_cross_version
def test_start_notebook_with_template() -> None:
    template_name = "test_start_notebook_with_template"
    tpl.set_template(template_name, conf.fixtures_path("templates/ntsc.yaml"))

    with cmd.interactive_command(
        "notebook", "start", "--template", template_name, "--detach"
    ) as nb:
        assert "SHOULDBE=SET" in cmd.get_command_config("notebook", str(nb.task_id))


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_cross_version
def test_start_command_with_template() -> None:
    template_name = "test_start_command_with_template"
    tpl.set_template(template_name, conf.fixtures_path("templates/ntsc.yaml"))

    with cmd.interactive_command(
        "command", "run", "--template", template_name, "--detach", "sleep infinity"
    ) as command:
        assert "SHOULDBE=SET" in cmd.get_command_config("command", str(command.task_id))


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_cross_version
def test_start_shell_with_template() -> None:
    template_name = "test_start_shell_with_template"
    tpl.set_template(template_name, conf.fixtures_path("templates/ntsc.yaml"))

    with cmd.interactive_command(
        "shell", "start", "--template", template_name, "--detach"
    ) as shell:
        assert "SHOULDBE=SET" in cmd.get_command_config("shell", str(shell.task_id))


def assert_templates_equal(t1: bindings.v1Template, t2: bindings.v1Template) -> None:
    assert t1.name == t2.name
    assert t1.config == t2.config
    assert t1.workspaceId == t2.workspaceId


def setup_template_test(
    session: Optional[Session] = None,
    workspace_id: Optional[int] = None,
    name: str = "template",
) -> Tuple[Session, bindings.v1Template]:
    session = api_utils.determined_test_session() if session is None else session
    tpl = bindings.v1Template(
        name=api_utils.get_random_string(),
        config=conf.load_config(conf.fixtures_path(f"templates/{name}.yaml")),
        workspaceId=workspace_id if workspace_id is not None else 1,
    )

    # create
    resp = bindings.post_PostTemplate(session, body=tpl, template_name=tpl.name)
    assert_templates_equal(tpl, resp.template)
    return (session, tpl)


@pytest.mark.e2e_cpu
def test_create_template() -> None:
    setup_template_test()


@pytest.mark.e2e_cpu
def test_read_template() -> None:
    session, tpl = setup_template_test()

    # read
    resp = bindings.get_GetTemplate(session, templateName=tpl.name)
    assert_templates_equal(tpl, resp.template)


@pytest.mark.e2e_cpu
def test_update_template() -> None:
    session, tpl = setup_template_test()

    # update
    tpl.config["description"] = "updated description"
    resp = bindings.patch_PatchTemplateConfig(session, body=tpl.config, templateName=tpl.name)
    assert_templates_equal(tpl, resp.template)


@pytest.mark.e2e_cpu
def test_delete_template() -> None:
    session, tpl = setup_template_test()

    # delete
    bindings.delete_DeleteTemplate(session, templateName=tpl.name)
    with pytest.raises(errors.NotFoundException):
        bindings.get_GetTemplate(session, templateName=tpl.name)
        pytest.fail("template should have been deleted")
