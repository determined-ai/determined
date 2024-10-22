import enum
from typing import List, Optional

from determined.common import api
from determined.common.api import bindings


class TokenType(enum.Enum):
    # UNSPECIFIED is internal to the bound API and is not be exposed to the front end
    USER_SESSION = bindings.v1TokenType.USER_SESSION.name
    ACCESS_TOKEN = bindings.v1TokenType.ACCESS_TOKEN.name


class AccessToken:
    """
    A class representing a AccessToken object that contains user session token info and
    access token info.
    It can be obtained from :func:`determined.experimental.client.list_access_tokens`
    Attributes:
        session: HTTP request session.
        token_id: (int) The ID of the access token in user sessions table.
        user_id: (int) Unique ID for the user.
        expiry: (str) Timestamp expires at reported.
        created_at: (str) Timestamp created at reported.
        token_type: (TokenType) Token type of the token.
        revoked: (Mutable, Optional[bool]) The datetime when the token was revoked.
            Null if the token is still active.
        description: (Mutable, Optional[str]) Human-friendly description of token.

    Note:
        Mutable properties may be changed by methods that update these values either automatically
        (eg. `revoke_tokens`, `edit_tokens`) or explicitly with :meth:`reload()`.
    """

    def __init__(self, token_id: int, session: api.Session):
        self.token_id = token_id
        self._session = session

        self.user_id: Optional[int] = None
        self.expiry: Optional[str] = None
        self.created_at: Optional[str] = None
        self.token_type: Optional[TokenType] = None
        self.revoked: Optional[bool] = None
        self.description: Optional[str] = None

    def _hydrate(self, tokenInfo: bindings.v1TokenInfo) -> None:
        self.user_id = tokenInfo.userId
        self.expiry = tokenInfo.expiry
        self.created_at = tokenInfo.createdAt
        self.token_type = tokenInfo.tokenType
        self.revoked = tokenInfo.revoked if tokenInfo.revoked is not None else False
        self.description = tokenInfo.description if tokenInfo.description is not None else ""

    def reload(self) -> None:
        resp = bindings.get_GetAccessTokens(
            session=self._session, tokenIds=[self.id], showInactive=True
        ).tokenInfo
        self._hydrate(resp[0])

    def edit_token(self, desc) -> None:
        patch_token_description = bindings.v1PatchAccessTokenRequest(
            tokenId=self.token_id, description=desc
        )
        bindings.patch_PatchAccessToken(
            self._session, body=patch_token_description, tokenId=self.token_id
        )
        self.reload()

    def revoke_token(self) -> None:
        patch_revoke_token = bindings.v1PatchAccessTokenRequest(
            tokenId=self.token_id, description=None, setRevoked=True
        )
        bindings.patch_PatchAccessToken(
            self._session, body=patch_revoke_token, tokenId=self.token_id
        )
        self.reload()

    def to_json(self):
        return {
            "token_id": self.token_id,
            "user_id": self.user_id,
            "description": self.description,
            "created_at": self.created_at if self.created_at else None,
            "expiry": self.expiry if self.expiry else None,
            "revoked": self.revoked if self.revoked else None,
            "token_type": self.token_type.name
            if isinstance(self.token_type, enum.Enum)
            else self.token_type,
        }

    @classmethod
    def _from_bindings(
        cls, AccessToken_bindings: List[bindings.v1TokenInfo], session: api.Session
    ) -> "AccessToken | List[AccessToken]":
        assert len(AccessToken_bindings) > 0

        access_token_infos = []
        for binding in AccessToken_bindings:
            assert binding.token_id
            AccessTokenInfo = cls(session=session, token_id=binding.token_id)
            AccessTokenInfo._hydrate(binding)
            access_token_infos.append(AccessTokenInfo)

        # Return a single instance if only one tokenInfo is provided
        if len(access_token_infos) == 1:
            return access_token_infos

        # Otherwise, return the list of AccessToken instances
        return access_token_infos
