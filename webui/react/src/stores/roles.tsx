import { Loadable, Loaded, NotLoaded } from 'components/kit/utils/loadable';
import { listRoles } from 'services/api';
import { ListRolesParams } from 'services/types';
import { UserRole } from 'types';
import handleError from 'utils/error';
import { observable, WritableObservable } from 'utils/observable';

class RoleStore {
  #roles: WritableObservable<Loadable<UserRole[]>> = observable(NotLoaded);

  public readonly roles = this.#roles.readOnly();

  public fetch(params: ListRolesParams = { limit: 0 }, signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    listRoles(params, { signal: signal ?? canceler.signal })
      .then((response) => {
        this.#roles.set(Loaded(response));
        return response;
      })
      .catch(handleError);

    return () => canceler.abort();
  }

  public reset() {
    this.#roles.set(NotLoaded);
  }
}

export default new RoleStore();
