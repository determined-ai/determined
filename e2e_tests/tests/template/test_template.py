from typing import Optional

import pytest

from determined.common import api, util
from determined.common.api import bindings, errors
from tests import api_utils
from tests import command as cmd
from tests import config as conf
from tests import experiment as exp
from tests import template as tpl
from tests.cluster import test_rbac


@pytest.mark.e2e_cpu
def test_set_template() -> None:
    sess = api_utils.user_session()
    template_name = "test_set_template"
    template_path = conf.fixtures_path("templates/template.yaml")
    tpl.set_template(sess, template_name, template_path)
    config = util.yaml_safe_load(tpl.describe_template(sess, template_name))
    assert config == conf.load_config(template_path)


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_cross_version
def test_start_notebook_with_template() -> None:
    sess = api_utils.user_session()
    template_name = "test_start_notebook_with_template"
    tpl.set_template(sess, template_name, conf.fixtures_path("templates/ntsc.yaml"))

    with cmd.interactive_command(
        sess, ["notebook", "start", "--template", template_name, "--detach"]
    ) as nb:
        assert "SHOULDBE=SET" in cmd.get_command_config(sess, "notebook", str(nb.task_id))


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_cross_version
def test_start_command_with_template() -> None:
    sess = api_utils.user_session()
    template_name = "test_start_command_with_template"
    tpl.set_template(sess, template_name, conf.fixtures_path("templates/ntsc.yaml"))

    with cmd.interactive_command(
        sess, ["command", "run", "--template", template_name, "--detach", "sleep infinity"]
    ) as command:
        assert "SHOULDBE=SET" in cmd.get_command_config(sess, "command", str(command.task_id))


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_cpu_cross_version
def test_start_shell_with_template() -> None:
    sess = api_utils.user_session()
    template_name = "test_start_shell_with_template"
    tpl.set_template(sess, template_name, conf.fixtures_path("templates/ntsc.yaml"))

    with cmd.interactive_command(
        sess, ["shell", "start", "--template", template_name, "--detach"]
    ) as shell:
        assert "SHOULDBE=SET" in cmd.get_command_config(sess, "shell", str(shell.task_id))


def assert_templates_equal(t1: bindings.v1Template, t2: bindings.v1Template) -> None:
    assert t1.name == t2.name
    assert t1.config == t2.config
    assert t1.workspaceId == t2.workspaceId


def setup_template_test(
    sess: api.Session,
    workspace_id: Optional[int] = None,
    name: str = "template",
) -> bindings.v1Template:
    tpl = bindings.v1Template(
        name=api_utils.get_random_string(),
        config=conf.load_config(conf.fixtures_path(f"templates/{name}.yaml")),
        workspaceId=workspace_id if workspace_id is not None else 1,
    )

    # create
    resp = bindings.post_PostTemplate(sess, body=tpl, template_name=tpl.name)
    assert_templates_equal(tpl, resp.template)
    return tpl


@pytest.mark.e2e_cpu
def test_create_template() -> None:
    sess = api_utils.user_session()
    setup_template_test(sess)


@pytest.mark.e2e_cpu
def test_read_template() -> None:
    sess = api_utils.user_session()
    tpl = setup_template_test(sess)

    # read
    resp = bindings.get_GetTemplate(sess, templateName=tpl.name)
    assert_templates_equal(tpl, resp.template)


@pytest.mark.e2e_cpu
def test_update_template() -> None:
    sess = api_utils.user_session()
    tpl = setup_template_test(sess)

    # update
    tpl.config["description"] = "updated description"
    resp = bindings.patch_PatchTemplateConfig(sess, body=tpl.config, templateName=tpl.name)
    assert_templates_equal(tpl, resp.template)


@pytest.mark.e2e_cpu
def test_delete_template() -> None:
    sess = api_utils.user_session()
    tpl = setup_template_test(sess)

    # delete
    bindings.delete_DeleteTemplate(sess, templateName=tpl.name)
    with pytest.raises(errors.NotFoundException):
        bindings.get_GetTemplate(sess, templateName=tpl.name)
        pytest.fail("template should have been deleted")


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_rbac_template_create() -> None:
    with test_rbac.create_workspaces_with_users(
        [
            [  # can create
                (0, ["Editor"]),
                (1, ["WorkspaceAdmin"]),
            ],
            [  # cannot create
                (0, ["Viewer"]),
            ],
        ]
    ) as (workspaces, sessions):
        for sess in sessions.values():
            setup_template_test(sess, workspaces[0].id)
            with pytest.raises(errors.ForbiddenException):
                setup_template_test(sess, workspaces[1].id)


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_rbac_template_delete() -> None:
    admin = api_utils.admin_session()
    with test_rbac.create_workspaces_with_users(
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
    ) as (workspaces, sessions):
        for sess in sessions.values():
            tpl = setup_template_test(admin, workspaces[0].id)
            bindings.delete_DeleteTemplate(sess, templateName=tpl.name)

        tpl = setup_template_test(admin, workspaces[1].id)
        with pytest.raises(errors.ForbiddenException):
            bindings.delete_DeleteTemplate(sessions[0], templateName=tpl.name)
        with pytest.raises(errors.NotFoundException):
            bindings.delete_DeleteTemplate(sessions[1], templateName=tpl.name)


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_rbac_template_view() -> None:
    admin = api_utils.admin_session()
    with test_rbac.create_workspaces_with_users(
        [
            [  # can view
                (0, ["Editor"]),
                (1, ["WorkspaceAdmin"]),
            ],
            [],  # none can view
        ]
    ) as (workspaces, sessions):
        tpl0 = setup_template_test(admin, workspaces[0].id)
        tpl1 = setup_template_test(admin, workspaces[1].id)
        for sess in sessions.values():
            bindings.get_GetTemplate(sess, templateName=tpl0.name)
            with pytest.raises(errors.NotFoundException):
                bindings.get_GetTemplate(sess, templateName=tpl1.name)


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_rbac_template_patch_config() -> None:
    admin = api_utils.admin_session()
    with test_rbac.create_workspaces_with_users(
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
    ) as (workspaces, sessions):
        tpl0 = setup_template_test(admin, workspaces[0].id)
        tpl1 = setup_template_test(admin, workspaces[1].id)
        for sess in sessions.values():
            tpl0.config["description"] = "updated description"
            bindings.patch_PatchTemplateConfig(
                sess,
                body=tpl0.config,
                templateName=tpl0.name,
            )
            with pytest.raises(errors.ForbiddenException):
                bindings.patch_PatchTemplateConfig(
                    sess,
                    body=tpl1.config,
                    templateName=tpl1.name,
                )


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
@pytest.mark.parametrize("kind", conf.ALL_NTSC)
def test_rbac_template_ntsc_create(kind: api.NTSC_Kind) -> None:
    admin = api_utils.admin_session()
    with test_rbac.create_workspaces_with_users(
        [
            [
                (0, ["Editor"]),
                (1, ["WorkspaceAdmin"]),
            ],
            [],
        ]
    ) as (workspaces, sessions):
        tpl0 = setup_template_test(admin, workspaces[0].id, name="ntsc")
        tpl1 = setup_template_test(admin, workspaces[1].id, name="ntsc")

        experiment_id = None
        pid = bindings.post_PostProject(
            admin,
            body=bindings.v1PostProjectRequest(name="test", workspaceId=workspaces[0].id),
            workspaceId=workspaces[0].id,
        ).project.id
        experiment_id = exp.create_experiment(
            admin,
            conf.fixtures_path("no_op/single.yaml"),
            conf.fixtures_path("no_op"),
            ["--project_id", str(pid)],
        )

        for sess in sessions.values():
            api_utils.launch_ntsc(
                sess, workspaces[0].id, kind, exp_id=experiment_id, template=tpl0.name
            )
            e = None
            with pytest.raises(errors.APIException) as e:
                api_utils.launch_ntsc(
                    sess, workspaces[0].id, kind, exp_id=experiment_id, template=tpl1.name
                )
            assert e.value.status_code == 404, e.value.message


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_rbac_template_exp_create() -> None:
    admin = api_utils.admin_session()
    with test_rbac.create_workspaces_with_users(
        [
            [
                (0, ["Editor"]),
                (1, ["WorkspaceAdmin"]),
            ],
            [],
        ]
    ) as (workspaces, sessions):
        tpl0 = setup_template_test(admin, workspaces[0].id)
        tpl1 = setup_template_test(admin, workspaces[1].id)

        pid = bindings.post_PostProject(
            admin,
            body=bindings.v1PostProjectRequest(name="test", workspaceId=workspaces[0].id),
            workspaceId=workspaces[0].id,
        ).project.id

        for sess in sessions.values():
            exp.create_experiment(
                sess,
                conf.fixtures_path("no_op/single.yaml"),
                conf.fixtures_path("no_op"),
                ["--project_id", str(pid), "--template", tpl0.name],
            )
            proc = exp.maybe_create_experiment(
                sess,
                conf.fixtures_path("no_op/single.yaml"),
                conf.fixtures_path("no_op"),
                ["--project_id", str(pid), "--template", tpl1.name],
            )
            assert proc.returncode == 1
            assert "not found" in proc.stderr
