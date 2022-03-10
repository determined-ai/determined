import pytest
import yaml

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
    tpl.set_template(template_name, conf.fixtures_path("templates/template.yaml"))

    with cmd.interactive_command("notebook", "start", "--template", template_name, "--detach"):
        pass


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_cross_version
def test_start_command_with_template() -> None:
    template_name = "test_start_command_with_template"
    tpl.set_template(template_name, conf.fixtures_path("templates/template.yaml"))

    with cmd.interactive_command(
        "command", "run", "--template", template_name, "--detach", "sleep infinity"
    ):
        pass


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_cross_version
def test_start_shell_with_template() -> None:
    template_name = "test_start_shell_with_template"
    tpl.set_template(template_name, conf.fixtures_path("templates/template.yaml"))

    with cmd.interactive_command("shell", "start", "--template", template_name, "--detach"):
        pass
