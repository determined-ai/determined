import { Observable, observable, WritableObservable } from 'micro-observables';

import { Loadable, Loaded, NotLoaded } from 'components/kit/utils/loadable';
import { getPermissionsSummary } from 'services/api';
import { UserAssignment, UserRole } from 'types';
import handleError from 'utils/error';

import PollingStore from './polling';

class PermissionStore extends PollingStore {
  #myAssignments: WritableObservable<Loadable<UserAssignment[]>> = observable(NotLoaded);
  #myRoles: WritableObservable<Loadable<UserRole[]>> = observable(NotLoaded);

  public readonly permissions = Observable.select(
    [this.#myAssignments, this.#myRoles],
    (assignments, roles) => Loadable.all([assignments, roles]),
  );
  public readonly myAssignments = this.#myAssignments.readOnly();
  public readonly myRoles = this.#myRoles.readOnly();

  // On login, fetching my user's assignments and roles in one API call.
  public fetch(signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    getPermissionsSummary({ signal: signal ?? canceler.signal })
      .then(({ assignments, roles }) => {
        this.#myAssignments.set(Loaded(assignments));
        this.#myRoles.set(Loaded(roles));
      })
      .catch(handleError);

    return () => canceler.abort();
  }

  // On logout, clear old user roles and assignments until new user login.
  public reset(): void {
    this.#myAssignments.set(NotLoaded);
    this.#myRoles.set(NotLoaded);
  }

  protected async poll() {
    const { assignments, roles } = await getPermissionsSummary({ signal: this.canceler?.signal });
    this.#myAssignments.set(Loaded(assignments));
    this.#myRoles.set(Loaded(roles));
  }
}

export default new PermissionStore();
