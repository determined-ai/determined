import { useCallback } from 'react';

import { StoreActionType, useStoreDispatch } from 'contexts/Store';
import { getAgents, getInfo, getUsers } from 'services/api';

export const useFetchAgents = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getAgents({ signal: canceler.signal });
      storeDispatch({ type: StoreActionType.SetAgents, value: response });
    } catch (e) {}
  }, [ canceler, storeDispatch ]);
};

export const useFetchInfo = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getInfo({ signal: canceler.signal });
      storeDispatch({ type: StoreActionType.SetInfo, value: response });
    } catch (e) {}
  }, [ canceler, storeDispatch ]);
};

export const useFetchUsers = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const usersResponse = await getUsers({ signal: canceler.signal });
      storeDispatch({ type: StoreActionType.SetUsers, value: usersResponse });
    } catch (e) {}
  }, [ canceler, storeDispatch ]);
};
