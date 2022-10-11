import uuid
from typing import Any, Dict, List, Optional

from requests import Response

from determined.common import api
from determined.common.api import authentication, bindings


# All the creates should be in
class AgentUserGroup:
    def __init__(
        self,
        agent_uid: Optional[int],
        agent_gid: Optional[int],
        agent_user: Optional[str],
        agent_group: Optional[str],
    ):
        self.agent_uid = agent_uid
        self.agent_gid = agent_gid
        self.agent_user = agent_user
        self.agent_group = agent_group


class User:
    def __init__(
        self,
        user_id: int,
        username: str,
        admin: bool,
        session: api.Session,
    ):
        self.username = username
        self.admin = admin
        self.user_id = user_id
        self.active = True
        self.agent_uid = None
        self.agent_gid = None
        self.agent_user = None
        self.agent_group = None
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
        admin=Optional[bool],
    ) -> Response:
        # new API -> bindings.patch_PatchUser(self.user_id, patchUser)
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
            admin=admin,
            agentUserGroup=v1agent_user_group,
        )
        resp = bindings.patch_PatchUser(self.session, userId=self.user_id, body=patch_user)
        # return API response
        return resp

    def update_username(self, new_username: str) -> Response:
        # return API response
        # API: bindings.patch_PatchUser(self.userid, patchUser) API (need to add username to message PatchUser in user.proto)
        patch_user = bindings.v1PatchUser(username=new_username)
        resp = bindings.patch_PatchUser(self.session, userId=self.user_id, body=patch_user)
        # return API response
        return resp
        pass

    def activate(self) -> Response:
        # calls update_user with active = true
        # bindings.patch_PatchUser(self.userid, patchUser)
        patch_user = bindings.v1PatchUser(active=True)
        resp = bindings.patch_PatchUser(self.session, userId=self.user_id, body=patch_user)

        return resp

    def deactivate(self) -> Response:
        # calls update_user with active = false
        # bindings.patch_PatchUser(self.user_id, patchUser) API
        patch_user = bindings.v1PatchUser(active=False)
        resp = bindings.patch_PatchUser(self.session, userId=self.user_id, body=patch_user)
        return resp

    def change_password(self, new_password: str) -> Response:
        # can also get user from authentication.must_cli_auth().get_session_user()
        # API bindings.patch_PatchUser need to add password to message PatchUser in user.proto
        patch_user = bindings.v1PatchUser(password=new_password)
        resp = bindings.patch_PatchUser(self.session, userId=self.user_id, body=patch_user)
        # return API response
        return resp

    def link_with_agent(self, agent_gid, agent_group, agent_uid, agent_user) -> Response:
        # calls update user with these args wrapped in agent_user_group.
        v1agent_user_group = bindings.v1AgentUserGroup(
            agentGid=agent_gid,
            agentGroup=agent_group,
            agentUid=agent_uid,
            agentUser=agent_user,
        )
        patch_user = bindings.v1PatchUser(agentUserGroup=v1agent_user_group)
        resp = bindings.patch_PatchUser(self.session, userId=self.user_id, body=patch_user)
        return resp
