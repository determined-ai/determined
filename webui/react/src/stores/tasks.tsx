import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';

import { getActiveTasks } from 'services/api';
import { TaskCounts } from 'types';
import { deepObservable } from 'utils/observable';

import PollingStore from './polling';

class TaskStore extends PollingStore {
  #activeTasks = deepObservable<Loadable<TaskCounts>>(NotLoaded);

  public readonly activeTasks = this.#activeTasks.readOnly();

  protected async poll() {
    const response = await getActiveTasks({}, { signal: this.canceler?.signal });
    this.#activeTasks.set(Loaded(response));
  }
}

export default new TaskStore();
