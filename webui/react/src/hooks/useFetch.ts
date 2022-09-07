import { useCallback } from 'react';

import { activeRunStates } from 'constants/states';
import { agentsToOverview, StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import {
  getActiveTasks,
  getAgents,
  getExperiments,
  getInfo,
  getResourcePools,
  getUsers,
  getUserSetting,
  getWorkspaces,
} from 'services/api';
import { ErrorType } from 'shared/utils/error';
import { BrandingType, ResourceType } from 'types';
import { updateFaviconType } from 'utils/browser';
import handleError from 'utils/error';

export const useFetchActiveExperiments = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getExperiments(
        { limit: -2, states: activeRunStates },
        { signal: canceler.signal },
      );
      storeDispatch({
        type: StoreAction.SetActiveExperiments,
        value: response.pagination.total || 0,
      });
    } catch (e) {
      handleError({
        message: 'Unable to fetch active experiments.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ canceler, storeDispatch ]);
};

export const useFetchAgents = (canceler: AbortController): () => Promise<void> => {
  const { info } = useStore();
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getAgents({ signal: canceler.signal });
      const cluster = agentsToOverview(response);
      storeDispatch({ type: StoreAction.SetAgents, value: response });
      updateFaviconType(
        cluster[ResourceType.ALL].allocation !== 0,
        info.branding || BrandingType.Determined,
      );
    } catch (e) { handleError(e); }
  }, [ canceler, info.branding, storeDispatch ]);
};

export const useFetchInfo = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getInfo({ signal: canceler.signal });
      storeDispatch({ type: StoreAction.SetInfo, value: response });
    } catch (e) {
      storeDispatch({ type: StoreAction.SetInfoCheck });
      handleError(e);
    }
  }, [ canceler, storeDispatch ]);
};

export const useFetchUsers = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const usersResponse = await getUsers({}, { signal: canceler.signal });
      storeDispatch({ type: StoreAction.SetUsers, value: usersResponse.users });
    } catch (e) { handleError(e); }
  }, [ canceler, storeDispatch ]);
};

export const useFetchUserSettings = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const userSettingResponse = await getUserSetting({}, { signal: canceler.signal });
      storeDispatch({ type: StoreAction.SetUserSettings, value: userSettingResponse.settings });
    } catch (e) { handleError(e); }
  }, [ canceler, storeDispatch ]);
};

export const useFetchResourcePools = (canceler?: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();
  return useCallback(async (): Promise<void> => {
    try {
      const resourcePools = await getResourcePools({}, { signal: canceler?.signal });
      storeDispatch({ type: StoreAction.SetResourcePools, value: resourcePools });
    } catch (e) { handleError(e); }
  }, [ canceler, storeDispatch ]);
};

export const useFetchActiveTasks = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const counts = await getActiveTasks({}, { signal: canceler.signal });
      storeDispatch({ type: StoreAction.SetActiveTasks, value: counts });
    } catch (e) {
      handleError({ message: 'Unable to fetch task counts.', silent: true, type: ErrorType.Api });
    }
  }, [ canceler, storeDispatch ]);
};

export const useFetchPinnedWorkspaces = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();
  return useCallback(async (): Promise<void> => {
    try {
      const pinnedWorkspaces = await getWorkspaces(
        { limit: 0, pinned: true },
        { signal: canceler.signal },
      );
      storeDispatch({ type: StoreAction.SetPinnedWorkspaces, value: pinnedWorkspaces.workspaces });
    } catch (e) { handleError(e); }
  }, [ canceler, storeDispatch ]);
};
