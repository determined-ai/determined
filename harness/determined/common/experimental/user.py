import uuid
from typing import Any, Dict, List, Optional

from requests import Response

from determined.common import api
from determined.common.api import authentication, bindings


class User:
    def __init__(
        self,
        user_id: int,
        username: str,
        admin: bool,
        session: api.Session,
        active: Optional[bool] = True,
        agent_uid: Optional[int] = None,
        agent_gid: Optional[int] = None,
        agent_user: Optional[str] = None,
        agent_group: Optional[str] = None,
    ):
        self.username = username
        self.admin = admin
        self.user_id = user_id
        self.active = active
        self.agent_uid = agent_uid
        self.agent_gid = agent_gid
        self.agent_user = agent_user
        self.agent_group = agent_group
        self.session = session

    def update(
        self,
        username: Optional[str] = None,
        active: Optional[bool] = None,
        password: Optional[str] = None,
        agent_uid: Optional[int] = None,
        agent_gid: Optional[int] = None,
        agent_user: Optional[str] = None,
        agent_group: Optional[str] = None,
        admin: Optional[bool] = None,
    ) -> Response:
        v1agent_user_group = bindings.v1AgentUserGroup(
            agentGid=agent_gid,
            agentGroup=agent_group,
            agentUid=agent_uid,
            agentUser=agent_user,
        )
        patch_user = bindings.v1PatchUser(
            username=username,
            password=password,
            active=active,
            agentUserGroup=v1agent_user_group
        )

        patch_user_req = bindings.v1PatchUserRequest(userId=self.user_id, user=patch_user)
        resp = bindings.patch_PatchUser(self.session,body=patch_user_req, userId=self.user_id)
        return resp

    def rename(self, new_username: str) -> Response:
        patch_user = bindings.v1PatchUser(username=new_username)
        patch_user_req = bindings.v1PatchUserRequest(userId=self.user_id, user=patch_user)
        resp = bindings.patch_PatchUser(self.session,body=patch_user_req, userId=self.user_id)
        return resp

    def activate(self) -> Response:
        patch_user = bindings.v1PatchUser(active=True)
        patch_user_req = bindings.v1PatchUserRequest(userId=self.user_id, user=patch_user)
        resp = bindings.patch_PatchUser(self.session,body=patch_user_req, userId=self.user_id)
        return resp

    def deactivate(self) -> Response:
        patch_user = bindings.v1PatchUser(active=False)
        patch_user_req = bindings.v1PatchUserRequest(userId=self.user_id, user=patch_user)
        resp = bindings.patch_PatchUser(self.session,body=patch_user_req, userId=self.user_id)
        return resp

    def change_password(self, new_password: str, is_hashed: Optional[bool] = False) -> Response:
        patch_user = bindings.v1PatchUser(password=new_password)
        patch_user_req = bindings.v1PatchUserRequest(userId=self.user_id, user=patch_user,isHashed=is_hashed)
        resp = bindings.patch_PatchUser(self.session,body=patch_user_req, userId=self.user_id)
        return resp

    def link_with_agent(self, agent_gid, agent_group, agent_uid, agent_user) -> Response:
        v1agent_user_group = bindings.v1AgentUserGroup(
            agentGid=agent_gid,
            agentGroup=agent_group,
            agentUid=agent_uid,
            agentUser=agent_user,
        )
        patch_user = bindings.v1PatchUser(agentUserGroup=v1agent_user_group)
        patch_user_req = bindings.v1PatchUserRequest(userId=self.user_id, user=patch_user)
        resp = bindings.patch_PatchUser(self.session,body=patch_user_req, userId=self.user_id)
        return resp
