import { Map } from 'immutable';
import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getExperiments } from 'services/api';
import { V1Pagination } from 'services/api-ts-sdk';
import { GetExperimentsParams } from 'services/types';
import { encodeParams } from 'utils/store';
import { ExperimentItem } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type ExperimentPagination = {
  experiments: ExperimentItem[];
  pagination?: V1Pagination;
};

type ExperimentsContext = {
  experimentsCache: Map<string, ExperimentPagination>;
  updateExperimentsCache: (ws: Map<string, ExperimentPagination>) => void;
};

const ExperimentsContext = createContext<ExperimentsContext | null>(null);

export const ExperimentsProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [experimentsCache, updateExperimentsCache] = useState<Map<string, ExperimentPagination>>(
    Map<string, ExperimentPagination>(),
  );
  return (
    <ExperimentsContext.Provider value={{ experimentsCache, updateExperimentsCache }}>
      {children}
    </ExperimentsContext.Provider>
  );
};

export const useFetchExperiments = (
  params: GetExperimentsParams,
  canceler: AbortController,
): (() => Promise<void>) => {
  const context = useContext(ExperimentsContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchExperiments outside of Experiment Context');
  }
  const { experimentsCache, updateExperimentsCache } = context;

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getExperiments(params, { signal: canceler.signal });
      updateExperimentsCache(experimentsCache.set(encodeParams(params), response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, experimentsCache, params, updateExperimentsCache]);
};

export const useExperiments = (params: GetExperimentsParams): Loadable<ExperimentPagination> => {
  const context = useContext(ExperimentsContext);
  if (context === null) {
    throw new Error('Attempted to use useExperiments outside of Experiment Context');
  }
  const loadedVal = context.experimentsCache.get(encodeParams(params));

  return loadedVal ? Loaded(loadedVal) : NotLoaded;
};
