import { useCallback } from 'react';

import { getAgents } from 'services/api';

import { StoreActionType, useStoreDispatch } from './Store';

export const useFetchAgents = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getAgents({ signal: canceler.signal });
      storeDispatch({ type: StoreActionType.SetAgents, value: response });
    } catch (e) {}
  }, [ canceler, storeDispatch ]);
};
