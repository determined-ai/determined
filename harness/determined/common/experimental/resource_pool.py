from typing import Any, Dict, List, Optional, Sequence

from determined.common import api
from determined.common.api import bindings
from determined.common.experimental import workspace


class ResourcePool:
    """A class representing a resource pool object.

    Attributes:
        name: (str) The name of the resource pool.
    """

    def __init__(
        self,
        session: api.Session,
        name: str,
        num_agents: Optional[int] = None,
        slots_per_agent: Optional[int] = None,
        slots_available: Optional[int] = None,
        slots_used: Optional[int] = None,
        slot_type: Optional[str] = None,
        accelerator: Optional[str] = None,
        default_compute_pool: Optional[bool] = None,
        default_aux_pool: Optional[bool] = None,
    ):
        """Create a resource pool object.

        Arguments:
            session: The session to use for API calls.
            name: (Optional) Name of the resource pool.
            num_agents: (Optional) The number of agents running in the resource pool.
            slots_per_agent: (Optional) The number of slots that exists on an dynamic agent.
            slots_available: (Optional) The total number of slots that exist in the resource pool.
            slots_used: (Optional) The number of slots that are actively running workloads.
            slot_type: (Optional) Slot device type: cpu, gpu, ...
            accelerator: (Optional) GCP, AWS accelerator information.
            default_compute_pool: (Optional) The default compute resource pool.
            default_aux_pool: (Optional) The default auxiliary resource pool.
        """
        self._session = session
        self.name = name
        self.num_agents = num_agents
        self.slots_per_agent = slots_per_agent
        self.slots_available = slots_available
        self.slots_used = slots_used
        self.slot_type = slot_type
        self.accelerator = accelerator
        self.default_compute_pool = default_compute_pool
        self.default_aux_pool = default_aux_pool
        if name is not None:
            self.bound_workspaces = self.list_workspaces()

    @classmethod
    def _from_bindings(
        cls, resource_pool_bindings: bindings.v1ResourcePool, session: api.Session
    ) -> "ResourcePool":
        resource_pool = cls(
            session,
            name=resource_pool_bindings.name,
            num_agents=resource_pool_bindings.numAgents,
            slots_per_agent=resource_pool_bindings.slotsPerAgent,
            slots_available=resource_pool_bindings.slotsAvailable,
            slots_used=resource_pool_bindings.slotsUsed,
            slot_type=resource_pool_bindings.slotType.value,
            accelerator=resource_pool_bindings.accelerator,
            default_compute_pool=resource_pool_bindings.defaultComputePool,
            default_aux_pool=resource_pool_bindings.defaultAuxPool,
        )
        return resource_pool

    def to_json(self) -> Dict[str, Any]:
        """JSONified representation of a ResourcePool"""
        rp_dict = self.__dict__
        del rp_dict["_session"]
        return rp_dict

    def add_bindings(
        self,
        workspace_names: List[str],
    ) -> None:
        """Binds a resource pool to one or more workspaces.

        A resource pool with bindings can only be used by workspaces bound to it. Attempting to add
        a binding that already exists results or binding workspaces or resource pools
        that do not exist will result in errors.
        """
        req = bindings.v1BindRPToWorkspaceRequest(
            resourcePoolName=self.name,
            workspaceNames=workspace_names,
        )

        bindings.post_BindRPToWorkspace(self._session, body=req, resourcePoolName=self.name)

    def remove_bindings(
        self,
        workspace_names: List[str],
    ) -> None:
        """Unbinds a resource pool from one or more workspaces.

        A resource pool with bindings can only be used by workspaces bound to it. Attempting to
        remove a binding that does not exist results in a no-op.
        """
        req = bindings.v1UnbindRPFromWorkspaceRequest(
            resourcePoolName=self.name,
            workspaceNames=workspace_names,
        )

        bindings.delete_UnbindRPFromWorkspace(self._session, body=req, resourcePoolName=self.name)

    def list_workspaces(self) -> Sequence[str]:
        """Lists the workspaces bound to a specified resource pool.

        A resource pool with bindings can only be used by workspaces bound to it.
        """

        def get_with_offset(offset: int) -> bindings.v1ListWorkspacesBoundToRPResponse:
            return bindings.get_ListWorkspacesBoundToRP(
                session=self._session,
                offset=offset,
                resourcePoolName=self.name,
            )

        resps = api.read_paginated(get_with_offset)

        workspace_names = []
        for r in resps:
            if r.workspaceIds is not None:
                workspace_names = [
                    workspace.Workspace(session=self._session, workspace_id=w).name
                    for w in r.workspaceIds
                ]

        return workspace_names

    def replace_bindings(
        self,
        workspace_names: List[str],
    ) -> None:
        """Replaces all the workspaces bound to a resource pool with those specified.

        If no bindings exist, new bindings will be added. Binding the same workspace more than once
        results in an SQL error. Binding workspaces or resource pools that do not exist result in
        Not Found errors.
        """
        req = bindings.v1OverwriteRPWorkspaceBindingsRequest(
            resourcePoolName=self.name,
            workspaceNames=workspace_names,
        )

        bindings.put_OverwriteRPWorkspaceBindings(
            self._session, body=req, resourcePoolName=self.name
        )


def list_resource_pools(session: api.Session) -> List[ResourcePool]:
    """List resource pools available to the cluster."""

    def get_with_offset(offset: int) -> bindings.v1GetResourcePoolsResponse:
        return bindings.get_GetResourcePools(
            session=session,
            offset=offset,
        )

    resps = api.read_paginated(get_with_offset)

    resource_pools = []
    for resp in resps:
        if resp.resourcePools is not None:
            for rp_bindings in resp.resourcePools:
                rp = ResourcePool._from_bindings(rp_bindings, session)
                resource_pools.append(rp)

    return resource_pools
