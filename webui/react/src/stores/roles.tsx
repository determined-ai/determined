import { listRoles } from 'services/api';
import { UserRole } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { observable, useObservable, WritableObservable } from 'utils/observable';

export class RolesService {
  static #roles: WritableObservable<Loadable<UserRole[]>> = observable(NotLoaded);

  static fetchRoles = async (canceler: AbortController): Promise<void> => {
    try {
      const response = await listRoles({ limit: 0 }, { signal: canceler.signal });
      this.#roles.set(Loaded(response));
    } catch (e) {
      handleError(e);
    }
  };

  static useRoles = (): Loadable<UserRole[]> => {
    return useObservable(this.#roles.readOnly());
  };
}
