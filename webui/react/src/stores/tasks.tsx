import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getActiveTasks } from 'services/api';
import { TaskCounts } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type TasksContext = {
  tasks: Loadable<TaskCounts>;
  updateTasks: (fn: (ws: Loadable<TaskCounts>) => Loadable<TaskCounts>) => void;
};

const TasksContext = createContext<TasksContext | null>(null);

export const TasksProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [state, setState] = useState<Loadable<TaskCounts>>(NotLoaded);
  return (
    <TasksContext.Provider value={{ tasks: state, updateTasks: setState }}>
      {children}
    </TasksContext.Provider>
  );
};

export const useFetchTasks = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(TasksContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchTasks outside of Task Context');
  }
  const { updateTasks } = context;

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getActiveTasks({}, { signal: canceler.signal });
      updateTasks(() => Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateTasks]);
};

export const useEnsureActiveTasksFetched = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(TasksContext);
  if (context === null) {
    throw new Error('Attempted to use useEnsureActiveTasksFetched outside of Task Context');
  }
  const { tasks, updateTasks } = context;

  return useCallback(async (): Promise<void> => {
    if (tasks !== NotLoaded) return;
    try {
      const response = await getActiveTasks({}, { signal: canceler.signal });
      updateTasks(() => Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, tasks, updateTasks]);
};

export const useTasks = (): Loadable<TaskCounts> => {
  const context = useContext(TasksContext);
  if (context === null) {
    throw new Error('Attempted to use useTasks outside of Task Context');
  }
  const { tasks } = context;

  return tasks;
};
