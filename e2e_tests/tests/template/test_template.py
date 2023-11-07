from typing import Optional, Tuple

import pytest

from determined.common import util
from determined.common.api import NTSC_Kind, Session, bindings, errors
from tests import api_utils
from tests import command as cmd
from tests import config as conf
from tests import experiment as exp
from tests import template as tpl
from tests.cluster import test_rbac as rbac
from tests.cluster import test_users as user


@pytest.mark.e2e_cpu
def test_set_template() -> None:
    template_name = "test_set_template"
    template_path = conf.fixtures_path("templates/template.yaml")
    tpl.set_template(template_name, template_path)
    config = util.yaml_safe_load(tpl.describe_template(template_name))
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


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac.rbac_disabled(), reason="ee rbac is required for this test")
def test_rbac_template_create() -> None:
    with rbac.create_workspaces_with_users(
        [
            [  # can create
                (0, ["Editor"]),
                (1, ["WorkspaceAdmin"]),
            ],
            [  # cannot create
                (0, ["Viewer"]),
            ],
        ]
    ) as (workspaces, creds):
        for uid in creds:
            setup_template_test(api_utils.determined_test_session(creds[uid]), workspaces[0].id)
            with pytest.raises(errors.ForbiddenException):
                setup_template_test(api_utils.determined_test_session(creds[uid]), workspaces[1].id)


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac.rbac_disabled(), reason="ee rbac is required for this test")
def test_rbac_template_delete() -> None:
    admin_session = api_utils.determined_test_session(conf.ADMIN_CREDENTIALS)
    with rbac.create_workspaces_with_users(
        [
            [  # can delete
                (0, ["Editor"]),
                (1, ["WorkspaceAdmin"]),
            ],
            [  # cannot delete
                (0, ["Viewer"]),
                (1, []),
            ],
        ]
    ) as (workspaces, creds):
        for uid in creds:
            _, tpl = setup_template_test(admin_session, workspaces[0].id)
            bindings.delete_DeleteTemplate(
                api_utils.determined_test_session(creds[uid]), templateName=tpl.name
            )

        _, tpl = setup_template_test(admin_session, workspaces[1].id)
        with pytest.raises(errors.ForbiddenException):
            bindings.delete_DeleteTemplate(
                api_utils.determined_test_session(creds[0]), templateName=tpl.name
            )
        with pytest.raises(errors.NotFoundException):
            bindings.delete_DeleteTemplate(
                api_utils.determined_test_session(creds[1]), templateName=tpl.name
            )


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac.rbac_disabled(), reason="ee rbac is required for this test")
def test_rbac_template_view() -> None:
    admin_session = api_utils.determined_test_session(conf.ADMIN_CREDENTIALS)
    with rbac.create_workspaces_with_users(
        [
            [  # can view
                (0, ["Editor"]),
                (1, ["WorkspaceAdmin"]),
            ],
            [],  # none can view
        ]
    ) as (workspaces, creds):
        _, tpl0 = setup_template_test(admin_session, workspaces[0].id)
        _, tpl1 = setup_template_test(admin_session, workspaces[1].id)
        for uid in creds:
            usession = api_utils.determined_test_session(creds[uid])
            bindings.get_GetTemplate(usession, templateName=tpl0.name)
            with pytest.raises(errors.NotFoundException):
                bindings.get_GetTemplate(usession, templateName=tpl1.name)


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac.rbac_disabled(), reason="ee rbac is required for this test")
def test_rbac_template_patch_config() -> None:
    admin_session = api_utils.determined_test_session(conf.ADMIN_CREDENTIALS)
    with rbac.create_workspaces_with_users(
        [
            [  # can update
                (0, ["Editor"]),
                (1, ["WorkspaceAdmin"]),
            ],
            [  # cannot update
                (0, ["Viewer"]),
                (1, ["Viewer"]),
            ],
        ]
    ) as (workspaces, creds):
        _, tpl0 = setup_template_test(admin_session, workspaces[0].id)
        _, tpl1 = setup_template_test(admin_session, workspaces[1].id)
        for uid in creds:
            tpl0.config["description"] = "updated description"
            bindings.patch_PatchTemplateConfig(
                api_utils.determined_test_session(creds[uid]),
                body=tpl0.config,
                templateName=tpl0.name,
            )
            with pytest.raises(errors.ForbiddenException):
                bindings.patch_PatchTemplateConfig(
                    api_utils.determined_test_session(creds[uid]),
                    body=tpl1.config,
                    templateName=tpl1.name,
                )


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac.rbac_disabled(), reason="ee rbac is required for this test")
@pytest.mark.parametrize("kind", conf.ALL_NTSC)
def test_rbac_template_ntsc_create(kind: NTSC_Kind) -> None:
    admin_session = api_utils.determined_test_session(conf.ADMIN_CREDENTIALS)
    with rbac.create_workspaces_with_users(
        [
            [
                (0, ["Editor"]),
                (1, ["WorkspaceAdmin"]),
            ],
            [],
        ]
    ) as (workspaces, creds):
        _, tpl0 = setup_template_test(admin_session, workspaces[0].id, name="ntsc")
        _, tpl1 = setup_template_test(admin_session, workspaces[1].id, name="ntsc")

        experiment_id = None
        pid = bindings.post_PostProject(
            admin_session,
            body=bindings.v1PostProjectRequest(name="test", workspaceId=workspaces[0].id),
            workspaceId=workspaces[0].id,
        ).project.id
        with user.logged_in_user(conf.ADMIN_CREDENTIALS):
            experiment_id = exp.create_experiment(
                conf.fixtures_path("no_op/single.yaml"),
                conf.fixtures_path("no_op"),
                ["--project_id", str(pid)],
            )

        for uid in creds:
            usession = api_utils.determined_test_session(creds[uid])
            api_utils.launch_ntsc(
                usession, workspaces[0].id, kind, exp_id=experiment_id, template=tpl0.name
            )
            e = None
            with pytest.raises(errors.APIException) as e:
                api_utils.launch_ntsc(
                    usession, workspaces[0].id, kind, exp_id=experiment_id, template=tpl1.name
                )
            assert e.value.status_code == 404, e.value.message


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac.rbac_disabled(), reason="ee rbac is required for this test")
def test_rbac_template_exp_create() -> None:
    admin_session = api_utils.determined_test_session(conf.ADMIN_CREDENTIALS)
    with rbac.create_workspaces_with_users(
        [
            [
                (0, ["Editor"]),
                (1, ["WorkspaceAdmin"]),
            ],
            [],
        ]
    ) as (workspaces, creds):
        _, tpl0 = setup_template_test(admin_session, workspaces[0].id)
        _, tpl1 = setup_template_test(admin_session, workspaces[1].id)

        pid = bindings.post_PostProject(
            admin_session,
            body=bindings.v1PostProjectRequest(name="test", workspaceId=workspaces[0].id),
            workspaceId=workspaces[0].id,
        ).project.id

        for uid in creds:
            with user.logged_in_user(creds[uid]):
                exp.create_experiment(
                    conf.fixtures_path("no_op/single.yaml"),
                    conf.fixtures_path("no_op"),
                    ["--project_id", str(pid), "--template", tpl0.name],
                )
                proc = exp.maybe_create_experiment(
                    conf.fixtures_path("no_op/single.yaml"),
                    conf.fixtures_path("no_op"),
                    ["--project_id", str(pid), "--template", tpl1.name],
                )
                assert proc.returncode == 1
                assert "not found" in proc.stderr
