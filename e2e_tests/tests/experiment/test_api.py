import uuid
from typing import Dict, List

import pytest

from determined.common.api import bindings
from tests import api_utils
from tests import experiment as exp
from tests.experiment import noop


@pytest.mark.e2e_cpu
def test_archived_proj_exp_list() -> None:
    admin = api_utils.admin_session()
    workspaces: List[bindings.v1Workspace] = []
    count = 2

    for _ in range(count):
        body = bindings.v1PostWorkspaceRequest(name=f"workspace_{uuid.uuid4().hex[:8]}")
        workspaces.append(bindings.post_PostWorkspace(admin, body=body).workspace)

    projects = []
    experiments = []
    experimentMap: Dict[int, List[int]] = {}
    for wrkspc in workspaces:
        workspace_projects = []
        for _ in range(count):
            proj_body = bindings.v1PostProjectRequest(
                name=f"p_{uuid.uuid4().hex[:8]}", workspaceId=wrkspc.id
            )
            pid = bindings.post_PostProject(
                admin,
                body=proj_body,
                workspaceId=wrkspc.id,
            ).project.id
            workspace_projects.append(pid)

        for p in workspace_projects:
            for _ in range(count):
                exp_ref = noop.create_paused_experiment(admin, p)
                experimentMap[p] = experimentMap.get(p, []) + [exp_ref.id]
                experiments.append(exp_ref.id)

        projects.extend(workspace_projects)

    for proj in experimentMap:
        bindings.post_KillExperiments(
            admin,
            body=bindings.v1KillExperimentsRequest(
                projectId=proj, experimentIds=experimentMap[proj]
            ),
            projectId=proj,
        )

    for x in experiments:
        exp.wait_for_experiment_state(admin, x, bindings.experimentv1State.CANCELED)

    archived_exp = [experiments[0], experiments[3], experiments[5], experiments[6]]

    for arch_exp in archived_exp:
        bindings.post_ArchiveExperiment(admin, id=arch_exp)

    # test1: GetExperiments shouldn't return archived experiments when archived flag is False
    r1 = bindings.get_GetExperiments(admin, archived=False)
    for e in r1.experiments:
        assert e.id not in archived_exp

    bindings.post_ArchiveProject(admin, id=projects[1])
    bindings.post_ArchiveProject(admin, id=projects[2])

    archived_exp.append(experiments[2])
    archived_exp.append(experiments[4])

    # test2: GetExperiments shouldn't return experiments from archived projects when
    # archived flag is False
    r2 = bindings.get_GetExperiments(admin, archived=False)
    for e in r2.experiments:
        assert e.id not in archived_exp

    bindings.post_ArchiveWorkspace(admin, id=workspaces[1].id)

    archived_exp.append(experiments[7])

    # test3: GetExperiments shouldn't return experiments from archived workspaces when
    # archived flag is False
    r3 = bindings.get_GetExperiments(admin, archived=False)
    for e in r3.experiments:
        assert e.id not in archived_exp

    # test4: GetExperiments should return only unarchived experiments within an
    # archived project when archived flag is false
    r4 = bindings.get_GetExperiments(admin, archived=False, projectId=projects[2])
    r4_correct_exp = [experiments[4]]
    assert len(r4.experiments) == len(r4_correct_exp)
    for e in r4.experiments:
        assert e.id in r4_correct_exp

    # test5: GetExperiments should return both archived and unarchived experiments when
    # archived flag is unspecified
    r5 = bindings.get_GetExperiments(admin)
    returned_e_id = []
    for e in r5.experiments:
        returned_e_id.append(e.id)

    for e_id in experiments:
        assert e_id in returned_e_id

    for w in workspaces:
        bindings.delete_DeleteWorkspace(admin, id=w.id)
