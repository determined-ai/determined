import { useCallback } from 'react';

import { StoreActionType, useStoreDispatch } from 'contexts/Store';
import { getInfo } from 'services/api';

export const useFetchInfo = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getInfo({ signal: canceler.signal });
      storeDispatch({ type: StoreActionType.SetInfo, value: response });
    } catch (e) {}
  }, [ canceler, storeDispatch ]);
};
