from typing import Dict

import pytest

import tests.config as conf
from determined.common import api
from determined.common.api import bindings
from determined.common.api._util import all_ntsc
from tests import api_utils
from tests import experiment as exp
from tests.cluster.test_rbac import create_workspaces_with_users, rbac_disabled
from tests.cluster.test_users import ADMIN_CREDENTIALS, det_run, logged_in_user


def seed_workspace(ws: bindings.v1Workspace) -> None:
    """set up each workspace with project, exp, and one of each ntsc"""
    admin_session = api_utils.determined_test_session(admin=True)
    pid = bindings.post_PostProject(
        admin_session,
        body=bindings.v1PostProjectRequest(name="test", workspaceId=ws.id),
        workspaceId=ws.id,
    ).project.id

    with logged_in_user(ADMIN_CREDENTIALS):
        print("creating experiment")
        experiment_id = exp.create_experiment(
            conf.fixtures_path("no_op/single.yaml"),
            conf.fixtures_path("no_op"),
            ["--project_id", str(pid)],
        )
    for ntsc in all_ntsc:
        print(f"creating {ntsc}")
        api_utils.launch_ntsc(admin_session, workspace_id=ws.id, typ=ntsc, exp_id=experiment_id)


@pytest.mark.e2e_cpu_rbac
@pytest.mark.skipif(rbac_disabled(), reason="ee rbac is required for this test")
def test_job_global_perm() -> None:
    with logged_in_user(ADMIN_CREDENTIALS):
        experiment_id = exp.create_experiment(
            conf.fixtures_path("no_op/single.yaml"),
            conf.fixtures_path("no_op"),
            ["--project_id", str(1)],
        )
        output = det_run(["job", "ls"])
        assert str(experiment_id) in str(output)


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
            ADMIN_CREDENTIALS: max_jobs,
            creds[0]: max_jobs,
            creds[1]: jobs_per_ws,
            creds[2]: jobs_per_ws,
        }

        workspace_ids = {ws.id for ws in workspaces}

        for cred, visible_count in expectations.items():
            v1_jobs = bindings.get_GetJobs(api_utils.determined_test_session(cred)).jobs
            # filterout jobs from other workspaces as the cluster is shared between tests
            v1_jobs = [j for j in v1_jobs if j.workspaceId in workspace_ids]
            assert len(v1_jobs) == visible_count, f"expected {visible_count} jobs for {cred}"

            jobs = bindings.get_GetJobsV2(api_utils.determined_test_session(cred)).jobs
            full_jobs = [
                j for j in jobs if j.full is not None and j.full.workspaceId in workspace_ids
            ]
            limited_jobs = [
                j for j in jobs if j.limited is not None and j.limited.workspaceId in workspace_ids
            ]
            assert len(limited_jobs) == max_jobs - visible_count
            assert len(full_jobs) == max_jobs - len(limited_jobs)
