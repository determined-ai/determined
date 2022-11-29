import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { activeRunStates } from 'constants/states';
import { getExperiments } from 'services/api';
import { ExperimentItem } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type ExperimentsContext = {
  experiments: Loadable<ExperimentItem[]>;
  updateExperiments: (fn: (ws: Loadable<ExperimentItem[]>) => Loadable<ExperimentItem[]>) => void;
};

const ExperimentsContext = createContext<ExperimentsContext | null>(null);

export const ExperimentsProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [state, setState] = useState<Loadable<ExperimentItem[]>>(NotLoaded);
  return (
    <ExperimentsContext.Provider value={{ experiments: state, updateExperiments: setState }}>
      {children}
    </ExperimentsContext.Provider>
  );
};

export const useFetchExperiments = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(ExperimentsContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchExperiments outside of Experiment Context');
  }
  const { updateExperiments } = context;

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getExperiments({}, { signal: canceler.signal });
      updateExperiments(() => Loaded(response.experiments));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateExperiments]);
};

export const useEnsureActiveExperimentsFetched = (
  canceler: AbortController,
): (() => Promise<void>) => {
  const context = useContext(ExperimentsContext);
  if (context === null) {
    throw new Error(
      'Attempted to use useEnsureActiveExperimentsFetched outside of Experiment Context',
    );
  }
  const { experiments, updateExperiments } = context;

  return useCallback(async (): Promise<void> => {
    if (experiments !== NotLoaded) return;
    try {
      const response = await getExperiments(
        { limit: -2, states: activeRunStates },
        { signal: canceler.signal },
      );
      updateExperiments(() => Loaded(response.experiments));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, experiments, updateExperiments]);
};

export const useExperiments = (): Loadable<ExperimentItem[]> => {
  const context = useContext(ExperimentsContext);
  if (context === null) {
    throw new Error('Attempted to use useExperiments outside of Experiment Context');
  }
  const { experiments } = context;

  return experiments;
};
