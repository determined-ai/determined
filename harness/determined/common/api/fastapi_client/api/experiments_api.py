# flake8: noqa E501
from asyncio import get_event_loop
from datetime import datetime
from typing import TYPE_CHECKING, Awaitable, List

from determined.common.api.fastapi_client import models as m
from determined.common.api.fapi import jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fastapi_client.api_client import ApiClient


class _ExperimentsApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_activate_experiment(self, id: int) -> Awaitable[m.Any]:
        path_params = {"id": str(id)}

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/experiments/{id}/activate",
            path_params=path_params,
        )

    def _build_for_determined_archive_experiment(self, id: int) -> Awaitable[m.Any]:
        path_params = {"id": str(id)}

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/experiments/{id}/archive",
            path_params=path_params,
        )

    def _build_for_determined_cancel_experiment(self, id: int) -> Awaitable[m.Any]:
        path_params = {"id": str(id)}

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/experiments/{id}/cancel",
            path_params=path_params,
        )

    def _build_for_determined_delete_experiment(self, experiment_id: int) -> Awaitable[m.Any]:
        path_params = {"experimentId": str(experiment_id)}

        return self.api_client.request(
            type_=m.Any,
            method="DELETE",
            url="/api/v1/experiments/{experimentId}",
            path_params=path_params,
        )

    def _build_for_determined_get_experiment(self, experiment_id: int) -> Awaitable[m.V1GetExperimentResponse]:
        path_params = {"experimentId": str(experiment_id)}

        return self.api_client.request(
            type_=m.V1GetExperimentResponse,
            method="GET",
            url="/api/v1/experiments/{experimentId}",
            path_params=path_params,
        )

    def _build_for_determined_get_experiment_checkpoints(
        self,
        id: int,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        validation_states: List[str] = None,
        states: List[str] = None,
    ) -> Awaitable[m.V1GetExperimentCheckpointsResponse]:
        path_params = {"id": str(id)}

        query_params = {}
        if sort_by is not None:
            query_params["sortBy"] = str(sort_by)
        if order_by is not None:
            query_params["orderBy"] = str(order_by)
        if offset is not None:
            query_params["offset"] = str(offset)
        if limit is not None:
            query_params["limit"] = str(limit)
        if validation_states is not None:
            query_params["validationStates"] = [
                str(validation_states_item) for validation_states_item in validation_states
            ]
        if states is not None:
            query_params["states"] = [str(states_item) for states_item in states]

        return self.api_client.request(
            type_=m.V1GetExperimentCheckpointsResponse,
            method="GET",
            url="/api/v1/experiments/{id}/checkpoints",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_determined_get_experiment_labels(
        self,
    ) -> Awaitable[m.V1GetExperimentLabelsResponse]:
        return self.api_client.request(
            type_=m.V1GetExperimentLabelsResponse,
            method="GET",
            url="/api/v1/experiment/labels",
        )

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

    def _build_for_determined_get_experiment_validation_history(
        self, experiment_id: int
    ) -> Awaitable[m.V1GetExperimentValidationHistoryResponse]:
        path_params = {"experimentId": str(experiment_id)}

        return self.api_client.request(
            type_=m.V1GetExperimentValidationHistoryResponse,
            method="GET",
            url="/api/v1/experiments/{experimentId}/validation-history",
            path_params=path_params,
        )

    def _build_for_determined_get_experiments(
        self,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        description: str = None,
        name: str = None,
        labels: List[str] = None,
        archived: bool = None,
        states: List[str] = None,
        users: List[str] = None,
    ) -> Awaitable[m.V1GetExperimentsResponse]:
        query_params = {}
        if sort_by is not None:
            query_params["sortBy"] = str(sort_by)
        if order_by is not None:
            query_params["orderBy"] = str(order_by)
        if offset is not None:
            query_params["offset"] = str(offset)
        if limit is not None:
            query_params["limit"] = str(limit)
        if description is not None:
            query_params["description"] = str(description)
        if name is not None:
            query_params["name"] = str(name)
        if labels is not None:
            query_params["labels"] = [str(labels_item) for labels_item in labels]
        if archived is not None:
            query_params["archived"] = str(archived)
        if states is not None:
            query_params["states"] = [str(states_item) for states_item in states]
        if users is not None:
            query_params["users"] = [str(users_item) for users_item in users]

        return self.api_client.request(
            type_=m.V1GetExperimentsResponse,
            method="GET",
            url="/api/v1/experiments",
            params=query_params,
        )

    def _build_for_determined_get_model_def(self, experiment_id: int) -> Awaitable[m.V1GetModelDefResponse]:
        path_params = {"experimentId": str(experiment_id)}

        return self.api_client.request(
            type_=m.V1GetModelDefResponse,
            method="GET",
            url="/api/v1/experiments/{experimentId}/model_def",
            path_params=path_params,
        )

    def _build_for_determined_get_trial(self, trial_id: int) -> Awaitable[m.V1GetTrialResponse]:
        path_params = {"trialId": str(trial_id)}

        return self.api_client.request(
            type_=m.V1GetTrialResponse,
            method="GET",
            url="/api/v1/trials/{trialId}",
            path_params=path_params,
        )

    def _build_for_determined_get_trial_checkpoints(
        self,
        id: int,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        validation_states: List[str] = None,
        states: List[str] = None,
    ) -> Awaitable[m.V1GetTrialCheckpointsResponse]:
        path_params = {"id": str(id)}

        query_params = {}
        if sort_by is not None:
            query_params["sortBy"] = str(sort_by)
        if order_by is not None:
            query_params["orderBy"] = str(order_by)
        if offset is not None:
            query_params["offset"] = str(offset)
        if limit is not None:
            query_params["limit"] = str(limit)
        if validation_states is not None:
            query_params["validationStates"] = [
                str(validation_states_item) for validation_states_item in validation_states
            ]
        if states is not None:
            query_params["states"] = [str(states_item) for states_item in states]

        return self.api_client.request(
            type_=m.V1GetTrialCheckpointsResponse,
            method="GET",
            url="/api/v1/trials/{id}/checkpoints",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_determined_kill_experiment(self, id: int) -> Awaitable[m.Any]:
        path_params = {"id": str(id)}

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/experiments/{id}/kill",
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

    def _build_for_determined_patch_experiment(
        self, experiment_id: int, body: m.V1Experiment
    ) -> Awaitable[m.V1PatchExperimentResponse]:
        path_params = {"experiment.id": str(experiment_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1PatchExperimentResponse,
            method="PATCH",
            url="/api/v1/experiments/{experiment.id}",
            path_params=path_params,
            json=body,
        )

    def _build_for_determined_pause_experiment(self, id: int) -> Awaitable[m.Any]:
        path_params = {"id": str(id)}

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/experiments/{id}/pause",
            path_params=path_params,
        )

    def _build_for_determined_preview_hp_search(
        self, body: m.V1PreviewHPSearchRequest
    ) -> Awaitable[m.V1PreviewHPSearchResponse]:
        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1PreviewHPSearchResponse, method="POST", url="/api/v1/preview-hp-search", json=body
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

    def _build_for_determined_unarchive_experiment(self, id: int) -> Awaitable[m.Any]:
        path_params = {"id": str(id)}

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/experiments/{id}/unarchive",
            path_params=path_params,
        )


class AsyncExperimentsApi(_ExperimentsApi):
    async def determined_activate_experiment(self, id: int) -> m.Any:
        return await self._build_for_determined_activate_experiment(id=id)

    async def determined_archive_experiment(self, id: int) -> m.Any:
        return await self._build_for_determined_archive_experiment(id=id)

    async def determined_cancel_experiment(self, id: int) -> m.Any:
        return await self._build_for_determined_cancel_experiment(id=id)

    async def determined_delete_experiment(self, experiment_id: int) -> m.Any:
        return await self._build_for_determined_delete_experiment(experiment_id=experiment_id)

    async def determined_get_experiment(self, experiment_id: int) -> m.V1GetExperimentResponse:
        return await self._build_for_determined_get_experiment(experiment_id=experiment_id)

    async def determined_get_experiment_checkpoints(
        self,
        id: int,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        validation_states: List[str] = None,
        states: List[str] = None,
    ) -> m.V1GetExperimentCheckpointsResponse:
        return await self._build_for_determined_get_experiment_checkpoints(
            id=id,
            sort_by=sort_by,
            order_by=order_by,
            offset=offset,
            limit=limit,
            validation_states=validation_states,
            states=states,
        )

    async def determined_get_experiment_labels(
        self,
    ) -> m.V1GetExperimentLabelsResponse:
        return await self._build_for_determined_get_experiment_labels()

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

    async def determined_get_experiment_validation_history(
        self, experiment_id: int
    ) -> m.V1GetExperimentValidationHistoryResponse:
        return await self._build_for_determined_get_experiment_validation_history(experiment_id=experiment_id)

    async def determined_get_experiments(
        self,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        description: str = None,
        name: str = None,
        labels: List[str] = None,
        archived: bool = None,
        states: List[str] = None,
        users: List[str] = None,
    ) -> m.V1GetExperimentsResponse:
        return await self._build_for_determined_get_experiments(
            sort_by=sort_by,
            order_by=order_by,
            offset=offset,
            limit=limit,
            description=description,
            name=name,
            labels=labels,
            archived=archived,
            states=states,
            users=users,
        )

    async def determined_get_model_def(self, experiment_id: int) -> m.V1GetModelDefResponse:
        return await self._build_for_determined_get_model_def(experiment_id=experiment_id)

    async def determined_get_trial(self, trial_id: int) -> m.V1GetTrialResponse:
        return await self._build_for_determined_get_trial(trial_id=trial_id)

    async def determined_get_trial_checkpoints(
        self,
        id: int,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        validation_states: List[str] = None,
        states: List[str] = None,
    ) -> m.V1GetTrialCheckpointsResponse:
        return await self._build_for_determined_get_trial_checkpoints(
            id=id,
            sort_by=sort_by,
            order_by=order_by,
            offset=offset,
            limit=limit,
            validation_states=validation_states,
            states=states,
        )

    async def determined_kill_experiment(self, id: int) -> m.Any:
        return await self._build_for_determined_kill_experiment(id=id)

    async def determined_kill_trial(self, id: int) -> m.Any:
        return await self._build_for_determined_kill_trial(id=id)

    async def determined_patch_experiment(
        self, experiment_id: int, body: m.V1Experiment
    ) -> m.V1PatchExperimentResponse:
        return await self._build_for_determined_patch_experiment(experiment_id=experiment_id, body=body)

    async def determined_pause_experiment(self, id: int) -> m.Any:
        return await self._build_for_determined_pause_experiment(id=id)

    async def determined_preview_hp_search(self, body: m.V1PreviewHPSearchRequest) -> m.V1PreviewHPSearchResponse:
        return await self._build_for_determined_preview_hp_search(body=body)

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

    async def determined_unarchive_experiment(self, id: int) -> m.Any:
        return await self._build_for_determined_unarchive_experiment(id=id)


class SyncExperimentsApi(_ExperimentsApi):
    def determined_activate_experiment(self, id: int) -> m.Any:
        coroutine = self._build_for_determined_activate_experiment(id=id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_archive_experiment(self, id: int) -> m.Any:
        coroutine = self._build_for_determined_archive_experiment(id=id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_cancel_experiment(self, id: int) -> m.Any:
        coroutine = self._build_for_determined_cancel_experiment(id=id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_delete_experiment(self, experiment_id: int) -> m.Any:
        coroutine = self._build_for_determined_delete_experiment(experiment_id=experiment_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_experiment(self, experiment_id: int) -> m.V1GetExperimentResponse:
        coroutine = self._build_for_determined_get_experiment(experiment_id=experiment_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_experiment_checkpoints(
        self,
        id: int,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        validation_states: List[str] = None,
        states: List[str] = None,
    ) -> m.V1GetExperimentCheckpointsResponse:
        coroutine = self._build_for_determined_get_experiment_checkpoints(
            id=id,
            sort_by=sort_by,
            order_by=order_by,
            offset=offset,
            limit=limit,
            validation_states=validation_states,
            states=states,
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_experiment_labels(
        self,
    ) -> m.V1GetExperimentLabelsResponse:
        coroutine = self._build_for_determined_get_experiment_labels()
        return get_event_loop().run_until_complete(coroutine)

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

    def determined_get_experiment_validation_history(
        self, experiment_id: int
    ) -> m.V1GetExperimentValidationHistoryResponse:
        coroutine = self._build_for_determined_get_experiment_validation_history(experiment_id=experiment_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_experiments(
        self,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        description: str = None,
        name: str = None,
        labels: List[str] = None,
        archived: bool = None,
        states: List[str] = None,
        users: List[str] = None,
    ) -> m.V1GetExperimentsResponse:
        coroutine = self._build_for_determined_get_experiments(
            sort_by=sort_by,
            order_by=order_by,
            offset=offset,
            limit=limit,
            description=description,
            name=name,
            labels=labels,
            archived=archived,
            states=states,
            users=users,
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_model_def(self, experiment_id: int) -> m.V1GetModelDefResponse:
        coroutine = self._build_for_determined_get_model_def(experiment_id=experiment_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_trial(self, trial_id: int) -> m.V1GetTrialResponse:
        coroutine = self._build_for_determined_get_trial(trial_id=trial_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_trial_checkpoints(
        self,
        id: int,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        validation_states: List[str] = None,
        states: List[str] = None,
    ) -> m.V1GetTrialCheckpointsResponse:
        coroutine = self._build_for_determined_get_trial_checkpoints(
            id=id,
            sort_by=sort_by,
            order_by=order_by,
            offset=offset,
            limit=limit,
            validation_states=validation_states,
            states=states,
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_kill_experiment(self, id: int) -> m.Any:
        coroutine = self._build_for_determined_kill_experiment(id=id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_kill_trial(self, id: int) -> m.Any:
        coroutine = self._build_for_determined_kill_trial(id=id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_patch_experiment(self, experiment_id: int, body: m.V1Experiment) -> m.V1PatchExperimentResponse:
        coroutine = self._build_for_determined_patch_experiment(experiment_id=experiment_id, body=body)
        return get_event_loop().run_until_complete(coroutine)

    def determined_pause_experiment(self, id: int) -> m.Any:
        coroutine = self._build_for_determined_pause_experiment(id=id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_preview_hp_search(self, body: m.V1PreviewHPSearchRequest) -> m.V1PreviewHPSearchResponse:
        coroutine = self._build_for_determined_preview_hp_search(body=body)
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

    def determined_unarchive_experiment(self, id: int) -> m.Any:
        coroutine = self._build_for_determined_unarchive_experiment(id=id)
        return get_event_loop().run_until_complete(coroutine)
