from typing import Optional


class Oauth2ScimClient:
    def __init__(
        self,
        client_id: str,
        domain: str,
        name: str,
        secret: Optional[str] = None,
    ):
        self.id = client_id
        self.secret = secret
        self.domain = domain
        self.name = name
