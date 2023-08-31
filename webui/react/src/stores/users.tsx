import { Map } from 'immutable';
import _ from 'lodash';

import { getCurrentUser, getUsers } from 'services/api';
import type { GetUsersParams } from 'services/types';
import { DetailedUser, DetailedUserList } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { observable, WritableObservable } from 'utils/observable';

import PollingStore from './polling';

function compareUser(a: DetailedUser, b: DetailedUser): number {
  const aName = a.displayName ?? a.username;
  const bName = b.displayName ?? b.username;
  return aName.localeCompare(bName);
}

class UserStore extends PollingStore {
  #userIds: WritableObservable<Loadable<number[]>> = observable(NotLoaded);
  #usersById: WritableObservable<Map<number, DetailedUser>> = observable(Map());
  #currentUser: WritableObservable<Loadable<DetailedUser>> = observable(NotLoaded);

  public readonly currentUser = this.#currentUser.readOnly();

  public getUser(id: number) {
    return this.#usersById.select((map) => {
      const user = map.get(id);
      return user ? Loaded(user) : NotLoaded;
    });
  }

  public getUsers() {
    return this.#userIds.select((loadable) => {
      const userIds = Loadable.getOrElse([], loadable);
      if (userIds.length === 0) return NotLoaded;

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
    let hasUserChanges = false;
    this.#usersById.update((prev) =>
      prev.withMutations((map) => {
        response.users.forEach((newUser) => {
          const oldUser = map.get(newUser.id);
          if (!_.isEqual(oldUser, newUser)) {
            map.set(newUser.id, newUser);
            hasUserChanges = true;
          }
        });
      }),
    );

    if (hasUserChanges) {
      this.#userIds.set(Loaded(response.users.map((user) => user.id)));
    }
  }
}

export default new UserStore();
