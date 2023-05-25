import uuid
from typing import List

import pytest

from determined.common.api import bindings
from determined.common.api.bindings import experimentv1State
from tests import api_utils
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_archived_proj_exp_list() -> None:
    session = api_utils.determined_test_session(admin=True)
    workspaces: List[bindings.v1Workspace] = []
    count = 2

    for _ in range(count):
        body = bindings.v1PostWorkspaceRequest(name=f"workspace_{uuid.uuid4().hex[:8]}")
        workspaces.append(bindings.post_PostWorkspace(session, body=body).workspace)

    projects = []
    experiments = []
    for wrkspc in workspaces:
        body1 = bindings.v1PostProjectRequest(
            name=f"p_{uuid.uuid4().hex[:8]}", workspaceId=wrkspc.id
        )
        pid1 = bindings.post_PostProject(
            session,
            body=body1,
            workspaceId=wrkspc.id,
        ).project.id

        body2 = bindings.v1PostProjectRequest(
            name=f"p_{uuid.uuid4().hex[:8]}", workspaceId=wrkspc.id
        )
        pid2 = bindings.post_PostProject(
            session,
            body=body2,
            workspaceId=wrkspc.id,
        ).project.id

        projects.append(pid1)
        projects.append(pid2)

        experiments.append(
            exp.create_experiment(
                conf.fixtures_path("no_op/single.yaml"),
                conf.fixtures_path("no_op"),
                ["--project_id", str(pid1), ("--paused")],
            )
        )
        experiments.append(
            exp.create_experiment(
                conf.fixtures_path("no_op/single.yaml"),
                conf.fixtures_path("no_op"),
                ["--project_id", str(pid1), ("--paused")],
            )
        )
        experiments.append(
            exp.create_experiment(
                conf.fixtures_path("no_op/single.yaml"),
                conf.fixtures_path("no_op"),
                ["--project_id", str(pid2), ("--paused")],
            )
        )
        experiments.append(
            exp.create_experiment(
                conf.fixtures_path("no_op/single.yaml"),
                conf.fixtures_path("no_op"),
                ["--project_id", str(pid2), ("--paused")],
            )
        )

    bindings.post_KillExperiments(
        session, body=bindings.v1KillExperimentsRequest(experimentIds=experiments)
    )

    for x in experiments:
        exp.wait_for_experiment_state(experiment_id=x, target_state=experimentv1State.CANCELED)

    archived_exp = [experiments[0], experiments[3], experiments[5], experiments[6]]

    for arch_exp in archived_exp:
        bindings.post_ArchiveExperiment(session, id=arch_exp)

    # test1: GetExperiments shouldn't return archived experiments when archived flag is False
    r1 = bindings.get_GetExperiments(session, archived=False)
    for e in r1.experiments:
        assert e.id not in archived_exp

    bindings.post_ArchiveProject(session, id=projects[1])
    bindings.post_ArchiveProject(session, id=projects[2])

    archived_exp.append(experiments[2])
    archived_exp.append(experiments[4])

    # test2: GetExperiments shouldn't return experiements from archived projects when
    # archived flag is False
    r2 = bindings.get_GetExperiments(session, archived=False)
    for e in r2.experiments:
        assert e.id not in archived_exp

    bindings.post_ArchiveWorkspace(session, id=workspaces[1].id)

    archived_exp.append(experiments[7])

    # test3: GetExperiments shouldn't return experiements from archived workspaces when
    # archived flag is False
    r3 = bindings.get_GetExperiments(session, archived=False)
    for e in r3.experiments:
        assert e.id not in archived_exp

    # test4: GetExperiments should return only unarchived experiments within an
    # archived project when archived flag is false
    r4 = bindings.get_GetExperiments(session, archived=False, projectId=projects[2])
    r4_correct_exp = [experiments[4]]
    assert len(r4.experiments) == len(r4_correct_exp)
    for e in r4.experiments:
        assert e.id in r4_correct_exp

    # test5: GetExperiments should return both archived and unarchived experiments when
    # archived flag is unspecified
    r5 = bindings.get_GetExperiments(session)
    returned_e_id = []
    for e in r5.experiments:
        returned_e_id.append(e.id)

    for e_id in experiments:
        assert e_id in returned_e_id

    for w in workspaces:
        bindings.delete_DeleteWorkspace(session, id=w.id)
