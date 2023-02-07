import { observable, Observable, WritableObservable } from 'micro-observables';

import { getPermissionsSummary } from 'services/api';
import { UserAssignment, UserRole } from 'types';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

export class PermissionsStore {
  static #userAssignments: WritableObservable<Loadable<UserAssignment[]>> = observable(NotLoaded);
  static #userRoles: WritableObservable<Loadable<UserRole[]>> = observable(NotLoaded);

  // On login, fetching my user's assignments and roles in one API call.
  static fetchMyAssignmentsAndRoles(canceler: AbortController): () => Promise<void> {
    return async () => {
      const { assignments, roles } = await getPermissionsSummary({ signal: canceler.signal });
      this.#userAssignments.set(Loaded(assignments));
      this.#userRoles.set(Loaded(roles));
    };
  }

  // Return the userAssignments observable (receive with useObservable)
  static getMyAssignments(): Observable<Loadable<UserAssignment[]>> {
    return this.#userAssignments;
  }

  // Return the userRoles observable (receive with useObservable)
  static getMyRoles(): Observable<Loadable<UserRole[]>> {
    return this.#userRoles;
  }

  // On logout, clear old user roles and assignments until new user login.
  static resetMyAssignmentsAndRoles(): void {
    this.#userAssignments.set(NotLoaded);
    this.#userRoles.set(NotLoaded);
  }
}
