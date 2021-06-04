from typing import Optional

from determined.common import util
from determined.common.api import authentication


class Session:
    def __init__(self, master: Optional[str], user: Optional[str]):
        self._master = master or util.get_default_master_address()
        self._user = user

        # TODO: use a local Authentication rather than the cli's singleton.
        authentication.cli_auth = authentication.Authentication(
            self._master, self._user, try_reauth=True
        )
