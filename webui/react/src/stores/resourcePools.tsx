import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getResourcePools } from 'services/api';
import { ResourcePool } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type ResourcePoolsContext = {
  resourcePools: Loadable<ResourcePool[]>;
  updateResourcePools: (
    fn: (resourcePools: Loadable<ResourcePool[]>) => Loadable<ResourcePool[]>,
  ) => void;
};

const ResourcePoolsContext = createContext<ResourcePoolsContext | null>(null);

export const ResourcePoolsProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [state, setState] = useState<Loadable<ResourcePool[]>>(NotLoaded);
  return (
    <ResourcePoolsContext.Provider value={{ resourcePools: state, updateResourcePools: setState }}>
      {children}
    </ResourcePoolsContext.Provider>
  );
};

export const useFetchResourcePools = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(ResourcePoolsContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchResourcePools outside of ResourcePool Context');
  }
  const { updateResourcePools } = context;

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getResourcePools({}, { signal: canceler.signal });
      updateResourcePools(() => Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateResourcePools]);
};

export const useResourcePools = (): Loadable<ResourcePool[]> => {
  const context = useContext(ResourcePoolsContext);
  if (context === null) {
    throw new Error('Attempted to useResourcePools outside of Resource Pool Context');
  }
  const { resourcePools } = context;

  return resourcePools;
};
