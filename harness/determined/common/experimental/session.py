from typing import Optional

from determined.common import util
from determined.common.api import authentication as auth


class Session:
    def __init__(self, master: Optional[str], user: Optional[str]):
        self._master = master or util.get_default_master_address()
        self._user = user
        auth.initialize_session(self._master, self._user, try_reauth=True)
