class Oauth2ScimCient: 
    def __init__(
        self, 
        id: str, 
        secret: Optional[str] = None, 
        domain: str, 
        name: str
    ): 

        self.id = id
        self.secret = secret
        self.domain = domain 
        self.name = name 