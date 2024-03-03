import pytest

from determined.common import streams
from determined.common.api import bindings
from determined.common.streams import _client
from tests import api_utils


@pytest.mark.e2e_cpu
def test_client_connection() -> None:
    sess = api_utils.admin_session()
    ws = _client.LomondStreamWebSocket(sess)
    stream = _client.Stream(ws)
    syncId = "sync1"
    stream.subscribe(sync_id=syncId, projects=_client.ProjectSpec(workspace_id=1))
    event = next(stream)
    assert event == _client.Sync(syncId, False)

    event = next(stream)
    assert event.id == 1
    assert event.immutable == True
    event = next(stream)
    assert event == _client.Sync(syncId, True)


@pytest.mark.e2e_cpu
def test_client_connection() -> None:
    sess = api_utils.admin_session()
    ws = _client.LomondStreamWebSocket(sess)
    stream = _client.Stream(ws)

    syncId = "sync1"
    projectName = "streaming_project"
    newProjectName = "streaming_project_1"

    resp_w = bindings.post_PostWorkspace(
        sess, body=bindings.v1PostWorkspaceRequest(name="streaming_workspace")
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

    stream.subscribe(sync_id=syncId, projects=_client.ProjectSpec(workspace_id=w.id))
    event = next(stream)
    assert event == _client.Sync(syncId, False)
    event = next(stream)
    assert event.id == p.id
    assert event.name == projectName
    seq = event.seq
    event = next(stream)
    assert event == _client.Sync(syncId, True)

    bindings.patch_PatchProject(sess, body=bindings.v1PatchProject(name=newProjectName), id=p.id)
    event = next(stream)
    assert event.id == p.id
    assert event.name == newProjectName
    assert event.seq > seq

    bindings.delete_DeleteProject(sess, id=p.id)
    for event in stream:
        if isinstance(event, streams.wire.ProjectMsg):
            assert event.state == "DELETING"
        elif isinstance(event, streams.wire.ProjectsDeleted):
            assert event == streams.wire.ProjectsDeleted(str(p.id))
            break
        else:
            raise ValueError(f"Unexpected message from stream. {event}")
