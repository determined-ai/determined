import { Map } from 'immutable';

import { getCurrentUser, getUsers } from 'services/api';
import { V1Pagination } from 'services/api-ts-sdk';
import type { GetUsersParams } from 'services/types';
import { DetailedUser } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { observable, WritableObservable } from 'utils/observable';
import { encodeParams } from 'utils/store';

import PollingStore from './polling';

type UsersPagination = {
  pagination: V1Pagination;
  userIds: number[];
};

function compareUser(a: DetailedUser, b: DetailedUser): number {
  const aName = a.displayName ?? a.username;
  const bName = b.displayName ?? b.username;
  return aName.localeCompare(bName);
}

class UserStore extends PollingStore {
  #usersById: WritableObservable<Map<number, DetailedUser>> = observable(Map());
  #usersBySearch: WritableObservable<Map<string, UsersPagination>> = observable(Map());
  #currentUser: WritableObservable<Loadable<DetailedUser>> = observable(NotLoaded);

  public readonly currentUser = this.#currentUser.readOnly();

  public getUser(id: number) {
    return this.#usersById.select((map) => {
      const user = map.get(id);
      return user ? Loaded(user) : NotLoaded;
    });
  }

  public getUsers(params: GetUsersParams = {}) {
    return this.getLoadableUsersByParams(params);
  }

  protected getLoadableUsersByParams(params: GetUsersParams = {}) {
    return this.#usersBySearch.select((map) => {
      const userIds = map.get(encodeParams(params))?.userIds;
      if (!userIds) return NotLoaded;

      const users = userIds
        .map((id) => this.#usersById.get().get(id))
        .filter((user) => !!user) as DetailedUser[];
      return Loaded(users.sort(compareUser));
    });
  }

  public updateCurrentUser(currentUser: DetailedUser) {
    this.#currentUser.set(Loaded(currentUser));
  }

  public updateUsers = (users: DetailedUser | DetailedUser[]) => {
    this.#usersById.update((prev) =>
      prev.withMutations((map) => {
        const iterUsers = Array.isArray(users) ? users : [users];
        iterUsers.forEach((user) => {
          map.set(user.id, user);

          // Update current user if applicable.
          const currentUser = Loadable.getOrElse(undefined, this.#currentUser.get());
          if (currentUser?.id === user.id) this.#currentUser.set(Loaded(user));
        });
      }),
    );
  };

  public fetchCurrentUser(signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    getCurrentUser({ signal: signal ?? canceler.signal })
      .then((response) => {
        this.#currentUser.set(Loaded(response));
        this.#usersById.update((map) => map.set(response.id, response));
      })
      .catch((e) => handleError(e, { publicSubject: 'Unable to fetch current user.' }));

    return () => canceler.abort();
  }

  public fetchUsers(params: GetUsersParams = {}, signal?: AbortSignal) {
    const canceler = new AbortController();

    getUsers(params, { signal: signal ?? canceler.signal })
      .then((response) => {
        this.#usersBySearch.update((map) =>
          map.set(encodeParams(params), {
            pagination: response.pagination,
            userIds: response.users.map((user) => user.id),
          }),
        );
        this.#usersById.update((prev) =>
          prev.withMutations((map) => {
            response.users.forEach((user) => map.set(user.id, user));
          }),
        );
      })
      .catch((e) => handleError(e, { publicSubject: 'Unable to fetch users.' }));

    return () => canceler.abort();
  }

  public reset() {
    this.#currentUser.set(NotLoaded);
    this.#usersById.set(Map());
    this.#usersBySearch.set(Map());
  }

  protected async poll(params: GetUsersParams = {}) {
    const response = await getUsers(params, { signal: this.canceler?.signal });
    this.#usersBySearch.update((map) =>
      map.set(encodeParams(params), {
        pagination: response.pagination,
        userIds: response.users.map((user) => user.id),
      }),
    );
    this.#usersById.update((prev) =>
      prev.withMutations((map) => {
        response.users.forEach((user) => map.set(user.id, user));
      }),
    );
  }
}

export default new UserStore();
