import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';

import { getPermissionsSummary } from 'services/api';
import { PermissionsSummary } from 'types';
import handleError from 'utils/error';
import { DeepObservable, deepObservable } from 'utils/observable';

import PollingStore from './polling';

class PermissionStore extends PollingStore {
  #permissionSummary: DeepObservable<Loadable<PermissionsSummary>> = deepObservable(NotLoaded);

  public readonly permissions = this.#permissionSummary.select((p) =>
    Loadable.map(p, (ps) => [ps.assignments, ps.roles]),
  );
  public readonly myAssignments = this.#permissionSummary.select((p) =>
    Loadable.map(p, (ps) => ps.assignments),
  );
  public readonly myRoles = this.#permissionSummary.select((p) =>
    Loadable.map(p, (ps) => ps.roles),
  );

  // On login, fetching my user's assignments and roles in one API call.
  public fetch(signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    this.getPermissionsSummary(signal ?? canceler.signal).catch(handleError);

    return () => canceler.abort();
  }

  // On logout, clear old user roles and assignments until new user login.
  public reset(): void {
    this.#permissionSummary.set(NotLoaded);
  }

  protected async getPermissionsSummary(signal?: AbortSignal): Promise<void> {
    this.#permissionSummary.set(Loaded(await getPermissionsSummary({ signal })));
  }

  protected poll() {
    return this.getPermissionsSummary(this.canceler?.signal);
  }
}

export default new PermissionStore();
