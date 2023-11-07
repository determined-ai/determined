from typing import Dict

import pytest

import tests.config as conf
from determined.common import api
from determined.common.api import NTSC_Kind, bindings, errors
from tests import api_utils
from tests import experiment as exp
from tests.cluster import test_rbac as rbac
from tests.cluster.test_rbac import (
    create_users_with_gloabl_roles,
    create_workspaces_with_users,
    rbac_disabled,
)
from tests.cluster.test_users import det_run, logged_in_user


def seed_workspace(ws: bindings.v1Workspace) -> None:
    """set up each workspace with project, exp, and one of each ntsc"""
    admin_session = api_utils.determined_test_session(admin=True)
    pid = bindings.post_PostProject(
        admin_session,
        body=bindings.v1PostProjectRequest(name="test", workspaceId=ws.id),
        workspaceId=ws.id,
    ).project.id

    with logged_in_user(conf.ADMIN_CREDENTIALS):
        print("creating experiment")
        experiment_id = exp.create_experiment(
            conf.fixtures_path("no_op/single-very-many-long-steps.yaml"),
            conf.fixtures_path("no_op"),
            ["--project_id", str(pid)],
        )
        print(f"created experiment {experiment_id}")
    for kind in conf.ALL_NTSC:
        print(f"creating {kind}")
        ntsc = api_utils.launch_ntsc(
            admin_session, workspace_id=ws.id, typ=kind, exp_id=experiment_id
        )
        print(f"created {kind} {ntsc.id}")


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac_disabled(), reason="ee rbac is required for this test")
def test_job_global_perm() -> None:
    with logged_in_user(conf.ADMIN_CREDENTIALS):
        experiment_id = exp.create_experiment(
            conf.fixtures_path("no_op/single.yaml"),
            conf.fixtures_path("no_op"),
            ["--project_id", str(1)],
        )
        output = det_run(["job", "ls"])
        assert str(experiment_id) in str(output)


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(
    rbac.strict_q_control_disabled(),
    reason="ee, rbac, " + "and strict q control are required for this test",
)
def test_job_strict_q_control() -> None:
    [cadmin] = create_users_with_gloabl_roles([["ClusterAdmin"]])

    with create_workspaces_with_users(
        [
            [
                (0, ["Editor"]),
            ],
        ]
    ) as (workspaces, creds):
        session = api_utils.determined_test_session(creds[0])
        r = api_utils.launch_ntsc(session, typ=NTSC_Kind.command, workspace_id=workspaces[0].id)

        cases = [
            rbac.PermCase(creds[0], errors.ForbiddenException),
            rbac.PermCase(cadmin, None),
        ]

        def action(cred: api.authentication.Credentials) -> None:
            session = api_utils.determined_test_session(cred)
            bindings.post_UpdateJobQueue(
                session,
                body=bindings.v1UpdateJobQueueRequest(
                    updates=[
                        bindings.v1QueueControl(jobId=r.jobId, priority=3),
                    ]
                ),
            )

        rbac.run_permission_tests(action, cases)


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac_disabled(), reason="ee rbac is required for this test")
def test_job_filtering() -> None:
    with create_workspaces_with_users(
        [
            [
                (0, ["Viewer", "Editor"]),
                (1, ["Viewer"]),
            ],
            [
                (2, ["Viewer"]),
                (0, ["Viewer"]),
            ],
            [
                (0, ["Editor"]),
            ],
        ]
    ) as (workspaces, creds):
        for ws in workspaces:
            seed_workspace(ws)

        jobs_per_ws = 5
        max_jobs = jobs_per_ws * len(workspaces)
        expectations: Dict[api.authentication.Credentials, int] = {
            conf.ADMIN_CREDENTIALS: max_jobs,
            creds[0]: max_jobs,
            creds[1]: jobs_per_ws,
            creds[2]: jobs_per_ws,
        }

        workspace_ids = {ws.id for ws in workspaces}

        for cred, visible_count in expectations.items():
            v1_jobs = bindings.get_GetJobs(api_utils.determined_test_session(cred)).jobs
            # filterout jobs from other workspaces as the cluster is shared between tests
            v1_jobs = [j for j in v1_jobs if j.workspaceId in workspace_ids]
            assert (
                len(v1_jobs) == visible_count
            ), f"expected {visible_count} jobs for {cred}. {v1_jobs}"

            jobs = bindings.get_GetJobsV2(api_utils.determined_test_session(cred)).jobs
            full_jobs = [
                j for j in jobs if j.full is not None and j.full.workspaceId in workspace_ids
            ]
            limited_jobs = [
                j for j in jobs if j.limited is not None and j.limited.workspaceId in workspace_ids
            ]
            assert len(limited_jobs) == max_jobs - visible_count
            assert len(full_jobs) == max_jobs - len(limited_jobs)
