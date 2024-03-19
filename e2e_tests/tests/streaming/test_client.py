import random

import pytest

from determined.common import streams
from determined.common.api import bindings
from tests import api_utils


@pytest.mark.e2e_cpu
def test_client_connection() -> None:
    sess = api_utils.admin_session()
    ws = streams._client.LomondStreamWebSocket(sess)
    stream = streams._client.Stream(ws)
    syncId = "sync1"
    stream.subscribe(sync_id=syncId, projects=streams._client.ProjectSpec(workspace_id=1))
    event = next(stream)
    assert event == streams._client.Sync(syncId, False)

    event = next(stream)
    assert isinstance(event, streams.wire.ProjectMsg)
    assert event.id == 1
    assert event.immutable is True
    event = next(stream)
    assert event == streams._client.Sync(syncId, True)


@pytest.mark.e2e_cpu
def test_client_subscribe() -> None:
    sess = api_utils.admin_session()
    ws = streams._client.LomondStreamWebSocket(sess)
    stream = streams._client.Stream(ws)

    syncId = "sync1"
    projectName = "streaming_project"
    newProjectName = "streaming_project_1"
    modelName = "streaming_model"
    newModelName = "streaming_model_1"
    pSeq = 0
    mSeq = 0

    resp_w = bindings.post_PostWorkspace(
        sess, body=bindings.v1PostWorkspaceRequest(name=f"streaming_workspace_{random.random()}")
    )
    w = resp_w.workspace
    resp_p = bindings.post_PostProject(
        sess,
        body=bindings.v1PostProjectRequest(
            name=projectName,
            workspaceId=w.id,
        ),
        workspaceId=w.id,
    )
    p = resp_p.project
    resp_m = bindings.post_PostModel(
        sess, body=bindings.v1PostModelRequest(name=modelName, workspaceId=w.id)
    )
    m = resp_m.model

    stream.subscribe(
        sync_id=syncId,
        projects=streams._client.ProjectSpec(workspace_id=w.id),
        models=streams._client.ModelSpec(workspace_id=w.id),
    )
    event = next(stream)
    assert event == streams._client.Sync(syncId, False)
    findProject, findModel, finish = False, False, False
    for event in stream:
        if isinstance(event, streams.wire.ProjectMsg):
            assert event.id == p.id
            assert event.name == projectName
            pSeq = event.seq
            findProject = True
        if isinstance(event, streams.wire.ModelMsg):
            assert event.id == m.id
            assert event.name == modelName
            mSeq = event.seq
            findModel = True
        if event == streams._client.Sync(syncId, True):
            finish = True
            break
    assert (
        findProject and findModel and finish
    ), f"Project found: {findProject}\n Model found: {findModel}\n Sync finished: {finish}"

    bindings.patch_PatchProject(sess, body=bindings.v1PatchProject(name=newProjectName), id=p.id)
    event = next(stream)
    assert isinstance(event, streams.wire.ProjectMsg)
    assert event.id == p.id
    assert event.name == newProjectName
    assert event.seq > pSeq

    bindings.patch_PatchModel(
        sess, body=bindings.v1PatchModel(name=newModelName), modelName=modelName
    )
    event = next(stream)
    assert isinstance(event, streams.wire.ModelMsg)
    assert event.id == m.id
    assert event.name == newModelName
    assert event.seq > mSeq

    bindings.delete_DeleteProject(sess, id=p.id)
    deleted = False
    for event in stream:
        if isinstance(event, streams.wire.ProjectMsg):
            assert event.state == "DELETING"
        elif isinstance(event, streams.wire.ProjectsDeleted):
            assert event == streams.wire.ProjectsDeleted(str(p.id))
            deleted = True
            break
        else:
            raise ValueError(f"Unexpected message from stream. {event}")
    assert deleted

    bindings.delete_DeleteModel(sess, modelName=newModelName)
    deleted = False
    for event in stream:
        if isinstance(event, streams.wire.ModelsDeleted):
            assert event == streams.wire.ModelsDeleted(str(m.id))
            deleted = True
            break
        else:
            raise ValueError(f"Unexpected message from stream. {event}")
    assert deleted
