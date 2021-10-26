# flake8: noqa E501
from asyncio import get_event_loop
from datetime import datetime
from typing import TYPE_CHECKING, Awaitable, List

from determined.common.api.fastapi_client import models as m

if TYPE_CHECKING:
    from determined.common.api.fapi import ApiClient


class _TrialsApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_get_experiment_trials(
        self,
        experiment_id: int,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        states: List[str] = None,
    ) -> Awaitable[m.V1GetExperimentTrialsResponse]:
        path_params = {"experimentId": str(experiment_id)}

        query_params = {}
        if sort_by is not None:
            query_params["sortBy"] = str(sort_by)
        if order_by is not None:
            query_params["orderBy"] = str(order_by)
        if offset is not None:
            query_params["offset"] = str(offset)
        if limit is not None:
            query_params["limit"] = str(limit)
        if states is not None:
            query_params["states"] = [str(states_item) for states_item in states]

        return self.api_client.request(
            type_=m.V1GetExperimentTrialsResponse,
            method="GET",
            url="/api/v1/experiments/{experimentId}/trials",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_determined_get_trial(self, trial_id: int) -> Awaitable[m.V1GetTrialResponse]:
        path_params = {"trialId": str(trial_id)}

        return self.api_client.request(
            type_=m.V1GetTrialResponse,
            method="GET",
            url="/api/v1/trials/{trialId}",
            path_params=path_params,
        )

    def _build_for_determined_kill_trial(self, id: int) -> Awaitable[m.Any]:
        path_params = {"id": str(id)}

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/trials/{id}/kill",
            path_params=path_params,
        )

    def _build_for_determined_trial_logs(
        self,
        trial_id: int,
        limit: int = None,
        follow: bool = None,
        agent_ids: List[str] = None,
        container_ids: List[str] = None,
        rank_ids: List[int] = None,
        levels: List[str] = None,
        stdtypes: List[str] = None,
        sources: List[str] = None,
        timestamp_before: datetime = None,
        timestamp_after: datetime = None,
        order_by: str = None,
    ) -> Awaitable[m.StreamResultOfV1TrialLogsResponse]:
        path_params = {"trialId": str(trial_id)}

        query_params = {}
        if limit is not None:
            query_params["limit"] = str(limit)
        if follow is not None:
            query_params["follow"] = str(follow)
        if agent_ids is not None:
            query_params["agentIds"] = [str(agent_ids_item) for agent_ids_item in agent_ids]
        if container_ids is not None:
            query_params["containerIds"] = [str(container_ids_item) for container_ids_item in container_ids]
        if rank_ids is not None:
            query_params["rankIds"] = [str(rank_ids_item) for rank_ids_item in rank_ids]
        if levels is not None:
            query_params["levels"] = [str(levels_item) for levels_item in levels]
        if stdtypes is not None:
            query_params["stdtypes"] = [str(stdtypes_item) for stdtypes_item in stdtypes]
        if sources is not None:
            query_params["sources"] = [str(sources_item) for sources_item in sources]
        if timestamp_before is not None:
            query_params["timestampBefore"] = str(timestamp_before)
        if timestamp_after is not None:
            query_params["timestampAfter"] = str(timestamp_after)
        if order_by is not None:
            query_params["orderBy"] = str(order_by)

        return self.api_client.request(
            type_=m.StreamResultOfV1TrialLogsResponse,
            method="GET",
            url="/api/v1/trials/{trialId}/logs",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_determined_trial_logs_fields(
        self, trial_id: int, follow: bool = None
    ) -> Awaitable[m.StreamResultOfV1TrialLogsFieldsResponse]:
        path_params = {"trialId": str(trial_id)}

        query_params = {}
        if follow is not None:
            query_params["follow"] = str(follow)

        return self.api_client.request(
            type_=m.StreamResultOfV1TrialLogsFieldsResponse,
            method="GET",
            url="/api/v1/trials/{trialId}/logs/fields",
            path_params=path_params,
            params=query_params,
        )


class AsyncTrialsApi(_TrialsApi):
    async def determined_get_experiment_trials(
        self,
        experiment_id: int,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        states: List[str] = None,
    ) -> m.V1GetExperimentTrialsResponse:
        return await self._build_for_determined_get_experiment_trials(
            experiment_id=experiment_id, sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, states=states
        )

    async def determined_get_trial(self, trial_id: int) -> m.V1GetTrialResponse:
        return await self._build_for_determined_get_trial(trial_id=trial_id)

    async def determined_kill_trial(self, id: int) -> m.Any:
        return await self._build_for_determined_kill_trial(id=id)

    async def determined_trial_logs(
        self,
        trial_id: int,
        limit: int = None,
        follow: bool = None,
        agent_ids: List[str] = None,
        container_ids: List[str] = None,
        rank_ids: List[int] = None,
        levels: List[str] = None,
        stdtypes: List[str] = None,
        sources: List[str] = None,
        timestamp_before: datetime = None,
        timestamp_after: datetime = None,
        order_by: str = None,
    ) -> m.StreamResultOfV1TrialLogsResponse:
        return await self._build_for_determined_trial_logs(
            trial_id=trial_id,
            limit=limit,
            follow=follow,
            agent_ids=agent_ids,
            container_ids=container_ids,
            rank_ids=rank_ids,
            levels=levels,
            stdtypes=stdtypes,
            sources=sources,
            timestamp_before=timestamp_before,
            timestamp_after=timestamp_after,
            order_by=order_by,
        )

    async def determined_trial_logs_fields(
        self, trial_id: int, follow: bool = None
    ) -> m.StreamResultOfV1TrialLogsFieldsResponse:
        return await self._build_for_determined_trial_logs_fields(trial_id=trial_id, follow=follow)


class SyncTrialsApi(_TrialsApi):
    def determined_get_experiment_trials(
        self,
        experiment_id: int,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        states: List[str] = None,
    ) -> m.V1GetExperimentTrialsResponse:
        coroutine = self._build_for_determined_get_experiment_trials(
            experiment_id=experiment_id, sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, states=states
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_trial(self, trial_id: int) -> m.V1GetTrialResponse:
        coroutine = self._build_for_determined_get_trial(trial_id=trial_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_kill_trial(self, id: int) -> m.Any:
        coroutine = self._build_for_determined_kill_trial(id=id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_trial_logs(
        self,
        trial_id: int,
        limit: int = None,
        follow: bool = None,
        agent_ids: List[str] = None,
        container_ids: List[str] = None,
        rank_ids: List[int] = None,
        levels: List[str] = None,
        stdtypes: List[str] = None,
        sources: List[str] = None,
        timestamp_before: datetime = None,
        timestamp_after: datetime = None,
        order_by: str = None,
    ) -> m.StreamResultOfV1TrialLogsResponse:
        coroutine = self._build_for_determined_trial_logs(
            trial_id=trial_id,
            limit=limit,
            follow=follow,
            agent_ids=agent_ids,
            container_ids=container_ids,
            rank_ids=rank_ids,
            levels=levels,
            stdtypes=stdtypes,
            sources=sources,
            timestamp_before=timestamp_before,
            timestamp_after=timestamp_after,
            order_by=order_by,
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_trial_logs_fields(
        self, trial_id: int, follow: bool = None
    ) -> m.StreamResultOfV1TrialLogsFieldsResponse:
        coroutine = self._build_for_determined_trial_logs_fields(trial_id=trial_id, follow=follow)
        return get_event_loop().run_until_complete(coroutine)
