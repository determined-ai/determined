from typing import Optional


class Oauth2ScimCient:
    def __init__(
        self,
        id: str,
        domain: str,
        name: str,
        secret: Optional[str] = None,
    ):

        self.id = id
        self.secret = secret
        self.domain = domain
        self.name = name
