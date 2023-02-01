import { observable, Observable, WritableObservable } from 'micro-observables';
import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getActiveTasks } from 'services/api';
import { TaskCounts } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

export class ActiveTasksService {
  static #activeTasks: WritableObservable<Loadable<TaskCounts>> = observable(NotLoaded);

  // Fetch counts to update the store.
  static updateActiveTasks(canceler: AbortController): () => Promise<void> {
    return async () => {
      const response = await getActiveTasks({}, { signal: canceler.signal });
      this.#activeTasks.set(Loaded(response));
    };
  }

  // Return the counts of active tasks via observable (receive with useObservable)
  static getTaskCounts(): Observable<Loadable<TaskCounts>> {
    return this.#activeTasks;
  }
}
