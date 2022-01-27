import { useCallback } from 'react';

import { agentsToOverview, StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import { getAgents, getInfo, getResourcePools, getUsers } from 'services/api';
import { ResourceType } from 'types';
import { updateFaviconType } from 'utils/browser';
import handleError from 'utils/error';

export const useFetchAgents = (canceler: AbortController): () => Promise<void> => {
  const { info } = useStore();
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getAgents({ signal: canceler.signal });
      const cluster = agentsToOverview(response);
      storeDispatch({ type: StoreAction.SetAgents, value: response });
      updateFaviconType(cluster[ResourceType.ALL].allocation !== 0, info.branding);
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
      const usersResponse = await getUsers({ signal: canceler.signal });
      storeDispatch({ type: StoreAction.SetUsers, value: usersResponse });
    } catch (e) { handleError(e); }
  }, [ canceler, storeDispatch ]);
};

export const useFetchResourcePools = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();
  return useCallback(async (): Promise<void> => {
    try {
      const resourcePools = await getResourcePools({}, { signal: canceler.signal });
      storeDispatch({ type: StoreAction.SetResourcePools, value: resourcePools });
    } catch (e) { handleError(e); }
  }, [ canceler, storeDispatch ]);
};
