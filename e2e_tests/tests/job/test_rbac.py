from typing import Callable, Dict, List

import pytest

import tests.config as conf
from determined.common import api
from determined.common.api import bindings, errors
from tests import api_utils, detproc
from tests import experiment as exp
from tests.cluster import test_rbac


def seed_workspace(ws: bindings.v1Workspace) -> None:
    """set up each workspace with project, exp, and one of each ntsc"""
    admin = api_utils.admin_session()
    pid = bindings.post_PostProject(
        admin,
        body=bindings.v1PostProjectRequest(name="test", workspaceId=ws.id),
        workspaceId=ws.id,
    ).project.id

    print("creating experiment")
    experiment_id = exp.create_experiment(
        admin,
        conf.fixtures_path("no_op/single-very-many-long-steps.yaml"),
        conf.fixtures_path("no_op"),
        ["--project_id", str(pid)],
    )
    print(f"created experiment {experiment_id}")

    for kind in conf.ALL_NTSC:
        print(f"creating {kind}")
        ntsc = api_utils.launch_ntsc(admin, workspace_id=ws.id, typ=kind, exp_id=experiment_id)
        print(f"created {kind} {ntsc.id}")


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_job_global_perm() -> None:
    admin = api_utils.admin_session()
    experiment_id = exp.create_experiment(
        admin,
        conf.fixtures_path("no_op/single.yaml"),
        conf.fixtures_path("no_op"),
        ["--project_id", str(1)],
    )
    output = detproc.check_output(admin, ["det", "job", "ls"])
    assert str(experiment_id) in output


def run_permission_tests(
    action: Callable[[api.Session], None], cases: List[test_rbac.PermCase]
) -> None:
    for cred, raises in cases:
        if raises is None:
            action(cred)
        else:
            with pytest.raises(raises):
                action(cred)


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_strict_q_control_not_enabled()
def test_job_strict_q_control() -> None:
    admin = api_utils.admin_session()
    cadmin, _ = api_utils.create_test_user()
    api_utils.assign_user_role(
        session=admin,
        user=cadmin.username,
        role="ClusterAdmin",
        workspace=None,
    )

    with test_rbac.create_workspaces_with_users(
        [
            [
                (0, ["Editor"]),
            ],
        ]
    ) as (workspaces, creds):
        r = api_utils.launch_ntsc(
            creds[0], typ=api.NTSC_Kind.command, workspace_id=workspaces[0].id
        )

        cases = [
            test_rbac.PermCase(creds[0], errors.ForbiddenException),
            test_rbac.PermCase(cadmin, None),
        ]

        def action(sess: api.Session) -> None:
            bindings.post_UpdateJobQueue(
                sess,
                body=bindings.v1UpdateJobQueueRequest(
                    updates=[
                        bindings.v1QueueControl(jobId=r.jobId, priority=3),
                    ]
                ),
            )

        run_permission_tests(action, cases)


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_job_filtering() -> None:
    with test_rbac.create_workspaces_with_users(
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
        admin = api_utils.admin_session()
        expectations: Dict[api.Session, int] = {
            admin: max_jobs,
            creds[0]: max_jobs,
            creds[1]: jobs_per_ws,
            creds[2]: jobs_per_ws,
        }

        workspace_ids = {ws.id for ws in workspaces}

        for cred, visible_count in expectations.items():
            v1_jobs = bindings.get_GetJobs(cred).jobs
            # filterout jobs from other workspaces as the cluster is shared between tests
            v1_jobs = [j for j in v1_jobs if j.workspaceId in workspace_ids]
            assert (
                len(v1_jobs) == visible_count
            ), f"expected {visible_count} jobs for {cred}. {v1_jobs}"

            jobs = bindings.get_GetJobsV2(cred).jobs
            full_jobs = [
                j for j in jobs if j.full is not None and j.full.workspaceId in workspace_ids
            ]
            limited_jobs = [
                j for j in jobs if j.limited is not None and j.limited.workspaceId in workspace_ids
            ]
            assert len(limited_jobs) == max_jobs - visible_count
            assert len(full_jobs) == max_jobs - len(limited_jobs)
