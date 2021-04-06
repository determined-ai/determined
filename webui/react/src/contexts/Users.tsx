import { useCallback } from 'react';

import { getUsers } from 'services/api';

import { StoreActionType, useStoreDispatch } from './Store';

export const useFetchUsers = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const usersResponse = await getUsers({ signal: canceler.signal });
      storeDispatch({ type: StoreActionType.SetUsers, value: usersResponse });
    } catch (e) {}
  }, [ canceler, storeDispatch ]);
};
