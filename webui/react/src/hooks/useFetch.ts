import { useCallback } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import { getWorkspaces, listRoles } from 'services/api';
import handleError from 'utils/error';

export const useFetchPinnedWorkspaces = (canceler: AbortController): (() => Promise<void>) => {
  const storeDispatch = useStoreDispatch();
  return useCallback(async (): Promise<void> => {
    try {
      const pinnedWorkspaces = await getWorkspaces(
        { limit: 0, pinned: true },
        { signal: canceler.signal },
      );
      storeDispatch({ type: StoreAction.SetPinnedWorkspaces, value: pinnedWorkspaces.workspaces });
    } catch (e) {
      handleError(e);
    }
  }, [canceler, storeDispatch]);
};

export const useFetchKnownRoles = (canceler: AbortController): (() => Promise<void>) => {
  const storeDispatch = useStoreDispatch();
  return useCallback(async (): Promise<void> => {
    try {
      const roles = await listRoles({ limit: 0 }, { signal: canceler.signal });
      storeDispatch({ type: StoreAction.SetKnownRoles, value: roles });
    } catch (e) {
      handleError(e);
    }
  }, [canceler, storeDispatch]);
};
