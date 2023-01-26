import { observable, Observable, WritableObservable } from 'micro-observables';

import { getPermissionsSummary } from 'services/api';
import { UserAssignment, UserRole } from 'types';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

export class UserRolesService {
  static #userAssignments: WritableObservable<Loadable<UserAssignment[]>> = observable(NotLoaded);
  static #userRoles: WritableObservable<Loadable<UserRole[]>> = observable(NotLoaded);

  static fetchUserAssignmentsAndRoles(canceler: AbortController): () => Promise<void> {
    return async () => {
      const { assignments, roles } = await getPermissionsSummary({ signal: canceler.signal });
      this.#userAssignments.set(Loaded(assignments));
      this.#userRoles.set(Loaded(roles));
    };
  }

  static getUserAssignments(): Observable<Loadable<UserAssignment[]>> {
    return this.#userAssignments;
  }

  static getUserRoles(): Observable<Loadable<UserRole[]>> {
    return this.#userRoles;
  }

  static resetUserAssignmentsAndRoles(): void {
    this.#userAssignments.set(NotLoaded);
    this.#userRoles.set(NotLoaded);
  }
}
