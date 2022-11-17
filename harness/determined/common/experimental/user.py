from typing import Optional

from determined.common import api
from determined.common.api import bindings


class User:
    def __init__(
        self,
        user_id: int,
        username: Optional[str],
        admin: Optional[bool],
        session: api.Session,
        active: Optional[bool] = True,
        display_name: Optional[str] = None,
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
        self._session = session
        self.display_name = display_name

    def _reload(self, raw: Optional[bindings.v1User] = None) -> None:
        if raw is None:
            raw = bindings.get_GetUser(session=self._session, userId=self.user_id).user
        assert raw.id is not None
        self.user_id = raw.id
        self.username = raw.username
        self.admin = raw.admin
        self.active = raw.active
        self.display_name = raw.displayName
        if raw.agentUserGroup is not None:
            self.agent_uid = raw.agentUserGroup.agentUid
            self.agent_gid = raw.agentUserGroup.agentGid
            self.agent_user = raw.agentUserGroup.agentUser
            self.agent_group = raw.agentUserGroup.agentGroup

    def rename(self, new_username: str) -> None:
        patch_user = bindings.v1PatchUser(username=new_username)
        resp = bindings.patch_PatchUser(self._session, body=patch_user, userId=self.user_id)
        self._reload(resp.user)

    def activate(self) -> None:
        patch_user = bindings.v1PatchUser(active=True)
        resp = bindings.patch_PatchUser(self._session, body=patch_user, userId=self.user_id)
        self._reload(resp.user)

    def deactivate(self) -> None:
        patch_user = bindings.v1PatchUser(active=False)
        resp = bindings.patch_PatchUser(self._session, body=patch_user, userId=self.user_id)
        self._reload(resp.user)

    def change_display_name(self, display_name: str) -> None:
        patch_user = bindings.v1PatchUser(displayName=display_name)
        resp = bindings.patch_PatchUser(self._session, body=patch_user, userId=self.user_id)
        self._reload(resp.user)

    def change_password(self, new_password: str) -> None:
        new_password = api.salt_and_hash(new_password)
        patch_user = bindings.v1PatchUser(password=new_password, isHashed=True)
        resp = bindings.patch_PatchUser(self._session, body=patch_user, userId=self.user_id)
        self._reload(resp.user)

    def link_with_agent(
        self,
        agent_uid: Optional[int] = None,
        agent_gid: Optional[int] = None,
        agent_user: Optional[str] = None,
        agent_group: Optional[str] = None,
    ) -> None:
        v1agent_user_group = bindings.v1AgentUserGroup(
            agentGid=agent_gid,
            agentGroup=agent_group,
            agentUid=agent_uid,
            agentUser=agent_user,
        )
        patch_user = bindings.v1PatchUser(agentUserGroup=v1agent_user_group)
        resp = bindings.patch_PatchUser(self._session, body=patch_user, userId=self.user_id)
        self._reload(resp.user)
