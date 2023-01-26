import { Observable, WritableObservable } from 'micro-observables';

import { getPermissionsSummary } from 'services/api';
import { UserAssignment, UserRole } from 'types';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

export class UserRolesService {
  static #isInternalConstructing = false;
  static #instance: UserRolesService;
  #userAssignments: WritableObservable<Loadable<UserAssignment[]>>;
  #userRoles: WritableObservable<Loadable<UserRole[]>>;

  private constructor() {
    if (!UserRolesService.#isInternalConstructing) {
      throw new TypeError('UserRolesService is not constructable');
    }
  }

  public static getInstance(): UserRolesService {
    if (!UserRolesService.#instance) {
      UserRolesService.#isInternalConstructing = true;
      UserRolesService.#instance = new UserRolesService();
      UserRolesService.#isInternalConstructing = false;
    }
    return UserRolesService.#instance;
  }

  public fetchUserAssignmentsAndRoles(canceler: AbortController): () => Promise<void> {
    return async () => {
      const { assignments, roles } = await getPermissionsSummary({ signal: canceler.signal });
      this.#userAssignments.set(Loaded(assignments));
      this.#userRoles.set(Loaded(roles));
    };
  }

  public getUserAssignments(): Observable<Loadable<UserAssignment[]>> {
    return this.#userAssignments;
  }

  public getUserRoles(): Observable<Loadable<UserRole[]>> {
    return this.#userRoles;
  }

  public resetUserAssignmentsAndRoles(): void {
    this.#userAssignments.set(NotLoaded);
    this.#userRoles.set(NotLoaded);
  }
}
