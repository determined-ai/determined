import { observable, WritableObservable } from 'micro-observables';

import { getActiveTasks } from 'services/api';
import { TaskCounts } from 'types';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

import PollingStore from './polling';

class TaskStore extends PollingStore {
  #activeTasks: WritableObservable<Loadable<TaskCounts>> = observable(NotLoaded);

  public readonly activeTasks = this.#activeTasks.readOnly();

  protected async poll() {
    const response = await getActiveTasks({}, { signal: this.canceler?.signal });
    this.#activeTasks.set(Loaded(response));
  }
}

export default new TaskStore();
