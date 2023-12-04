import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { Map } from 'immutable';
import { Observable } from 'micro-observables';

import { getCurrentUser, getUsers, patchUser } from 'services/api';
import type { GetUsersParams, PatchUserParams } from 'services/types';
import { DetailedUser, DetailedUserList } from 'types';
import { asValueObjectFactory, ValueObjectOf } from 'utils/asValueObject';
import handleError from 'utils/error';
import { deepObservable, immutableObservable } from 'utils/observable';

import PollingStore from './polling';

function compareUser(a: DetailedUser, b: DetailedUser): number {
  const aName = a.displayName ?? a.username;
  const bName = b.displayName ?? b.username;
  return aName.localeCompare(bName);
}

const asValueOfUser = asValueObjectFactory(DetailedUser);

class UserStore extends PollingStore {
  // TODO: investigate replacing userIds + usersById with OrderedMap
  #userIds = deepObservable<Loadable<number[]>>(NotLoaded);
  #usersById = immutableObservable<Map<number, ValueObjectOf<DetailedUser>>>(Map());
  #currentUser = deepObservable<Loadable<DetailedUser>>(NotLoaded);

  public readonly currentUser = this.#currentUser.readOnly();

  public getUser(id: number): Observable<Loadable<DetailedUser>> {
    return this.#usersById.select((map) => {
      const user = map.get(id);
      return user ? Loaded(user) : NotLoaded;
    });
  }

  public getUsers(): Observable<Loadable<DetailedUser[]>> {
    return Observable.select([this.#userIds, this.#usersById], (loadable, usersById) => {
      const userIds = Loadable.getOrElse([], loadable);
      if (userIds.length === 0) return NotLoaded;

      const users = userIds
        .map(usersById.get.bind(usersById))
        .filter((u): u is Exclude<typeof u, undefined> => u !== undefined)
        .sort(compareUser);

      return Loaded(users);
    });
  }

  public updateCurrentUser(currentUser: DetailedUser) {
    this.#currentUser.set(Loaded(currentUser));
  }

  public async patchUser(userId: number, userParams: PatchUserParams['userParams']) {
    const user = await patchUser({ userId, userParams });
    this.#usersById.update((prev) => {
      return prev.set(user.id, asValueOfUser(user));
    });
    const currentUser = Loadable.getOrElse(undefined, this.#currentUser.get());
    if (currentUser?.id === user.id) this.#currentUser.set(Loaded(user));
  }

  public fetchCurrentUser(signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    getCurrentUser({ signal: signal ?? canceler.signal })
      .then((response) => {
        this.#currentUser.set(Loaded(response));
        this.#usersById.update((map) => map.set(response.id, asValueOfUser(response)));
      })
      .catch((e) => handleError(e, { publicSubject: 'Unable to fetch current user.' }));

    return () => canceler.abort();
  }

  public fetchUsers(signal?: AbortSignal) {
    const canceler = new AbortController();

    getUsers({}, { signal: signal ?? canceler.signal })
      .then((response) => {
        this.updateUsersFromResponse(response);
      })
      .catch((e) => handleError(e, { publicSubject: 'Unable to fetch users.' }));

    return () => canceler.abort();
  }

  public reset() {
    this.#userIds.set(NotLoaded);
    this.#currentUser.set(NotLoaded);
    this.#usersById.set(Map());
  }

  protected async poll(params: GetUsersParams = {}) {
    const response = await getUsers(params, { signal: this.canceler?.signal });
    this.updateUsersFromResponse(response);
  }

  protected updateUsersFromResponse(response: DetailedUserList) {
    this.#usersById.update((prev) =>
      prev.withMutations((map) => {
        response.users.forEach((newUser) => {
          map.set(newUser.id, asValueOfUser(newUser));
        });
      }),
    );

    this.#userIds.set(Loaded(response.users.map((user) => user.id)));
  }
}

export default new UserStore();
