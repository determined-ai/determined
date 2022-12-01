import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getActiveTasks } from 'services/api';
import { TaskCounts } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type TasksContext = {
  activeTasks: Loadable<TaskCounts>;
  updateActiveTasks: (fn: (ws: Loadable<TaskCounts>) => Loadable<TaskCounts>) => void;
};

const TasksContext = createContext<TasksContext | null>(null);

export const TasksProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [state, setState] = useState<Loadable<TaskCounts>>(NotLoaded);
  return (
    <TasksContext.Provider value={{ activeTasks: state, updateActiveTasks: setState }}>
      {children}
    </TasksContext.Provider>
  );
};

export const useFetchActiveTasks = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(TasksContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchTasks outside of Task Context');
  }
  const { updateActiveTasks } = context;

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getActiveTasks({}, { signal: canceler.signal });
      updateActiveTasks(() => Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateActiveTasks]);
};

export const useEnsureActiveTasksFetched = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(TasksContext);
  if (context === null) {
    throw new Error('Attempted to use useEnsureActiveTasksFetched outside of Task Context');
  }
  const { activeTasks, updateActiveTasks } = context;

  return useCallback(async (): Promise<void> => {
    if (activeTasks !== NotLoaded) return;
    try {
      const response = await getActiveTasks({}, { signal: canceler.signal });
      updateActiveTasks(() => Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, activeTasks, updateActiveTasks]);
};

export const useActiveTasks = (): Loadable<TaskCounts> => {
  const context = useContext(TasksContext);
  if (context === null) {
    throw new Error('Attempted to use useActiveTasks outside of Task Context');
  }
  const { activeTasks } = context;

  return activeTasks;
};
