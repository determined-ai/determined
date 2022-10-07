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
        password: str,
        admin: bool,
        session: api.Session,
    ):
        self.username = username
        self.password = password
        self.admin = admin
        self.user_id = user_id
        self.active = True
        self.agent_uid = None
        self.agent_gid = None
        self.agent_user = None
        self.agent_group = None
        self.session = session

    def update(
        self, username: str,
        active: Optional[bool] = None,
        password: Optional[str] = None,
        agent_user_group: Optional[AgentUserGroup] = None,
    ) -> Response:
        # new API -> bindings.patch_PatchUser(self.user_id, patchUser)
        # return API response
        pass
     
    def update_username(self, new_username: str) -> Response:
        # return API response
        # API: bindings.patch_PatchUser(self.userid, patchUser) API (need to add username to message PatchUser in user.proto)

        pass

    def activate(self) -> None:
        # calls update_user with active = true
        # bindings.patch_PatchUser(self.userid, patchUser) 
        pass

    def deactivate(self) -> None:
        # calls update_user with active = false
        # bindings.patch_PatchUser(self.user_id, patchUser) API
        pass


    def change_password(self, new_password: str) -> None:
        # can also get user from authentication.must_cli_auth().get_session_user()
        # API bindings.patch_PatchUser need to add password to message PatchUser in user.proto
        pass

    def link_with_agent(self, agent_user_group: AgentUserGroup) -> None:
        # calls update user with these args wrapped in agent_user_group.
        pass