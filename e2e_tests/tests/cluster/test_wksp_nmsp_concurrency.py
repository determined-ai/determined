import uuid
from typing import Tuple

import pytest
import urllib3

from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests.task import task


# Workspace-namespace binding requests were causing a deadlock in our Kubernetes jobs handler when
# a thread that locks the jobsService tries to reacquire the lock during the execution of a callback
# function that only gets called when a job is running in the namespace that we want to bind to the
# workspace. Verify that we don't run into this deadlock when triggering multiple calls to the
# API handler that binds a workspace to an auto-created namespace, as this request can trigger the
# deadlock if the namespace (or verify the existence of for the first time) is running a job.
@pytest.mark.e2e_single_k8s
@pytest.mark.timeout(3 * 60)
def test_wksp_running_task_check_namespace(namespaces_created: Tuple[str, str]) -> None:
    sess = api_utils.admin_session()
    namespace, _ = namespaces_created
    wksp_namespace_meta = bindings.v1WorkspaceNamespaceMeta(
        namespace=namespace,
    )
    sess._max_retries = urllib3.util.retry.Retry(total=5, backoff_factor=0.5)
    cluster_name = conf.DEFAULT_RM_CLUSTER_NAME

    # Create a workspace bound to an auto-created namespace.
    body = bindings.v1PostWorkspaceRequest(name=f"workspace_{uuid.uuid4().hex[:8]}")
    body.clusterNamespaceMeta = {cluster_name: wksp_namespace_meta}
    resp = bindings.post_PostWorkspace(sess, body=body)
    wksp_id = resp.workspace.id
    notebook_id = bindings.post_LaunchNotebook(
        sess,
        body=bindings.v1LaunchNotebookRequest(workspaceId=wksp_id),
    ).notebook.id

    # Wait for the notebook to start or run.
    task.wait_for_task_start(sess, notebook_id, start_or_run=True)

    # Set a workspace-namespace binding using the same auto-created namespace.
    content = bindings.v1SetWorkspaceNamespaceBindingsRequest(workspaceId=wksp_id)
    namespace_meta = bindings.v1WorkspaceNamespaceMeta(
        namespace=namespace,
    )
    content.clusterNamespaceMeta = {cluster_name: namespace_meta}

    # Can we run this request repeatedly with no deadlock?
    for _ in range(3):
        bindings.post_SetWorkspaceNamespaceBindings(sess, body=content, workspaceId=wksp_id)

    # Cleanup.
    bindings.delete_DeleteWorkspace(sess, id=wksp_id)
