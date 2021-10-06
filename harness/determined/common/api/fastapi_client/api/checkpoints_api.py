# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable

from determined.common.api.fastapi_client import models as m
from determined.common.api.fapi_helper import to_json as jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fastapi_client.api_client import ApiClient


class _CheckpointsApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_get_checkpoint(self, checkpoint_uuid: str) -> Awaitable[m.V1GetCheckpointResponse]:
        path_params = {"checkpointUuid": str(checkpoint_uuid)}

        return self.api_client.request(
            type_=m.V1GetCheckpointResponse,
            method="GET",
            url="/api/v1/checkpoints/{checkpointUuid}",
            path_params=path_params,
        )

    def _build_for_determined_post_checkpoint_metadata(
        self, checkpoint_uuid: str, body: m.V1PostCheckpointMetadataRequest
    ) -> Awaitable[m.V1PostCheckpointMetadataResponse]:
        path_params = {"checkpoint.uuid": str(checkpoint_uuid)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1PostCheckpointMetadataResponse,
            method="POST",
            url="/api/v1/checkpoints/{checkpoint.uuid}/metadata",
            path_params=path_params,
            json=body,
        )


class AsyncCheckpointsApi(_CheckpointsApi):
    async def determined_get_checkpoint(self, checkpoint_uuid: str) -> m.V1GetCheckpointResponse:
        return await self._build_for_determined_get_checkpoint(checkpoint_uuid=checkpoint_uuid)

    async def determined_post_checkpoint_metadata(
        self, checkpoint_uuid: str, body: m.V1PostCheckpointMetadataRequest
    ) -> m.V1PostCheckpointMetadataResponse:
        return await self._build_for_determined_post_checkpoint_metadata(checkpoint_uuid=checkpoint_uuid, body=body)


class SyncCheckpointsApi(_CheckpointsApi):
    def determined_get_checkpoint(self, checkpoint_uuid: str) -> m.V1GetCheckpointResponse:
        coroutine = self._build_for_determined_get_checkpoint(checkpoint_uuid=checkpoint_uuid)
        return get_event_loop().run_until_complete(coroutine)

    def determined_post_checkpoint_metadata(
        self, checkpoint_uuid: str, body: m.V1PostCheckpointMetadataRequest
    ) -> m.V1PostCheckpointMetadataResponse:
        coroutine = self._build_for_determined_post_checkpoint_metadata(checkpoint_uuid=checkpoint_uuid, body=body)
        return get_event_loop().run_until_complete(coroutine)
