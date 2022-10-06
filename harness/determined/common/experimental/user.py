import uuid
from determined.common import api 
from determined.common.api import bindings
from determined.common.api import authentication
from typing import Any, Dict, List, Optional
from requests import Response

# All the creates should be in 
class User:
    def __init__(self, username, password, admin):
        self.username = username
        self.password = password
        self.admin = admin
        self.active = True 
        self.agent_uid = None
        self.agent_gid = None 
        self.agent_user = None
        self.agent_group = None
    
    def update_user(username: str, master_address: str, active: Optional[bool] = None,   password: Optional[str] = None,  agent_user_group: Optional[Dict[str, Any]] = None,) -> Response:
        # return API response
        pass

    def update_username(current_username: str, master_address: str, new_username: str) -> Response:
         # return API response
        pass 

    def activate_user(username: str, master_address: str) -> None: 
        #calls update_user with active = true 
        pass 

    def deactivate_user(username: str, master_address: str) -> None: 
        #calls update_user with active = false 
        pass 

    def log_in_user(username: str, password: str, master_address: str) -> None: 
          # for password should they pass in plain text or hashed value (applies to other methods too.)
          #  but how would we unhash it? 
        pass

    def log_out_user(username: str, master_address: str) -> None:
        pass 

    def rename(name_target_user: str, master_address: str, new_username: str) -> None: 
        pass 

    def change_password(new_password: str, username: Optional[str]) -> None: 
        # can get user from authentication.must_cli_auth().get_session_user()
        pass

    def link_with_agent_user(agent_uid: uuid, agent_user: str, agent_gid: int, agent_group: str) -> None: 
        # calls update user with these args wrapped in agent_user_group. 
        pass

    def whoami()-> str:
        # return username 
        pass
