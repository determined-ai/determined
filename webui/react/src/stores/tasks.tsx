import { observable, Observable, WritableObservable } from 'micro-observables';
import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getActiveTasks } from 'services/api';
import { TaskCounts } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

export class ActiveTasksStore {
  static #activeTasks: WritableObservable<TaskCounts> = observable({
    commands: 0,
    notebooks: 0,
    shells: 0,
    tensorboards: 0,
  });

  // Fetch counts to update the store.
  static updateActiveTasks(canceler: AbortController): () => Promise<void> {
    return async () => {
      const response = await getActiveTasks({}, { signal: canceler.signal });
      this.#activeTasks.set(response);
    };
  }

  // Return the counts of active tasks via observable (receive with useObservable)
  static getTaskCounts(): Observable<TaskCounts> {
    return this.#activeTasks;
  }
}
