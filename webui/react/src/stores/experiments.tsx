import { Map } from 'immutable';
import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getExperiments } from 'services/api';
import { V1Pagination } from 'services/api-ts-sdk';
import { GetExperimentsParams } from 'services/types';
import { ExperimentItem } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type ExperimentPagination = {
  experiments: ExperimentItem[];
  pagination?: V1Pagination;
};

type ExperimentsContext = {
  experimentsIndex: Map<string, ExperimentPagination>;
  updateExperimentsIndex: (ws: Map<string, ExperimentPagination>) => void;
};

const ExperimentsContext = createContext<ExperimentsContext | null>(null);

const encodeArgs = (args: { [key: string]: any }): string => {
  const orderedKeys = Object.keys(args).sort();
  return orderedKeys.map((key: string) => `${key}=${JSON.stringify(args[key])}`).join(',');
};

export const ExperimentsProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [experimentsIndex, updateExperimentsIndex] = useState<Map<string, ExperimentPagination>>(
    Map<string, ExperimentPagination>(),
  );
  return (
    <ExperimentsContext.Provider value={{ experimentsIndex, updateExperimentsIndex }}>
      {children}
    </ExperimentsContext.Provider>
  );
};

export const useFetchExperiments = (
  filters: GetExperimentsParams,
  canceler: AbortController,
): (() => Promise<void>) => {
  const context = useContext(ExperimentsContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchExperiments outside of Experiment Context');
  }
  const { experimentsIndex, updateExperimentsIndex } = context;

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getExperiments(filters, { signal: canceler.signal });
      updateExperimentsIndex(experimentsIndex.set(encodeArgs(filters), response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, experimentsIndex, filters, getExperiments, updateExperimentsIndex]);
};

export const useExperiments = (filters: GetExperimentsParams): Loadable<ExperimentPagination> => {
  const context = useContext(ExperimentsContext);
  if (context === null) {
    throw new Error('Attempted to use useExperiments outside of Experiment Context');
  }
  const loadedVal = context.experimentsIndex.get(encodeArgs(filters));

  return loadedVal ? Loaded(loadedVal) : NotLoaded;
};
