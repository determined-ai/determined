from typing import cast

from sgqlc import operation
from sgqlc.endpoint import http

from determined_common import check
from determined_common.api import gql, request


class GraphQLQueryError(ConnectionError):
    pass


class GraphQLQuery:
    def __init__(self, master: str) -> None:
        headers = request.add_token_to_headers({})
        master_url = request.make_url(master, "graphql")

        self.endpoint = http.HTTPEndpoint(master_url, base_headers=headers)
        self.op = operation.Operation(gql.query_root)

    def send(self) -> gql.query_root:
        resp = self.endpoint(self.op)
        errors = resp.get("errors")
        if errors:
            # The errors field, if present, is a list of dictionaries, each containing at least the
            # key "message" (see https://sgqlc.readthedocs.io/en/latest/sgqlc.endpoint.http.html).
            raise GraphQLQueryError("; ".join(e["message"] for e in errors))
        return cast(gql.query_root, self.op + resp)


def decode_bytes(s: str) -> str:
    r"""
    Hasura sends over any bytea value as the two-character string '\x' followed by the hex encoding
    of the bytes. This function turns such a value into the corresponding string.
    """
    check.true(s.startswith(r"\x"), "Invalid log value received")
    return bytes.fromhex(s[2:]).decode()
