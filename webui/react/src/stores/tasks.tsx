import { observable, Observable, WritableObservable } from 'micro-observables';

import { getActiveTasks } from 'services/api';
import { TaskCounts } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

export class TasksStore {
  static #activeTasks: WritableObservable<Loadable<TaskCounts>> = observable(NotLoaded);

  // Fetch counts to update the store.
  static async fetchActiveTasks(canceler: AbortController): Promise<void> {
    try {
      const response = await getActiveTasks({}, { signal: canceler.signal });
      this.#activeTasks.set(Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }

  // Return the counts of active tasks via observable (receive with useObservable)
  static getActiveTaskCounts(): Observable<Loadable<TaskCounts>> {
    return this.#activeTasks;
  }
}
