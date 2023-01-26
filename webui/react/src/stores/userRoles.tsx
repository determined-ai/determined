import { observable, Observable, WritableObservable } from 'micro-observables';

import { getPermissionsSummary } from 'services/api';
import { UserAssignment, UserRole } from 'types';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

export class UserRolesService {
  static #userAssignments: WritableObservable<Loadable<UserAssignment[]>> = observable(NotLoaded);
  static #userRoles: WritableObservable<Loadable<UserRole[]>> = observable(NotLoaded);

  // On login, fetching my user's assignments and roles in one API call.
  static fetchUserAssignmentsAndRoles(canceler: AbortController): () => Promise<void> {
    return async () => {
      const { assignments, roles } = await getPermissionsSummary({ signal: canceler.signal });
      this.#userAssignments.set(Loaded(assignments));
      this.#userRoles.set(Loaded(roles));
    };
  }

  // Return the userAssignments observable (receive with useObservable)
  static getUserAssignments(): Observable<Loadable<UserAssignment[]>> {
    return this.#userAssignments;
  }

  // Return the userRoles observable (receive with useObservable)
  static getUserRoles(): Observable<Loadable<UserRole[]>> {
    return this.#userRoles;
  }

  // On logout, clear old user roles and assignments until new user login.
  static resetUserAssignmentsAndRoles(): void {
    this.#userAssignments.set(NotLoaded);
    this.#userRoles.set(NotLoaded);
  }
}
