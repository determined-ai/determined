import { listRoles } from 'services/api';
import { UserRole } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { observable, useObservable, WritableObservable } from 'utils/observable';

export class KnownRolesService {
  static #knownRoles: WritableObservable<Loadable<UserRole[]>> = observable(NotLoaded);

  static fetchKnownRoles = async (canceler: AbortController): Promise<void> => {
    try {
      const response = await listRoles({ limit: 0 }, { signal: canceler.signal });
      this.#knownRoles.set(Loaded(response));
    } catch (e) {
      handleError(e);
    }
  };

  static useKnownRoles = (): Loadable<UserRole[]> => {
    return useObservable(this.#knownRoles.readOnly());
  };
}
