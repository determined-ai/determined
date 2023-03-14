import { Map } from 'immutable';

import { getCurrentUser, getUsers } from 'services/api';
import { V1Pagination } from 'services/api-ts-sdk';
import type { GetUsersParams as FetchUsersConfig } from 'services/types';
import { DetailedUser, DetailedUserList } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { Observable, observable, WritableObservable } from 'utils/observable';
import { encodeParams } from 'utils/store';

type UsersPagination = {
  pagination: V1Pagination;
  users: number[];
};

class UsersService {
  #users: WritableObservable<Map<number, DetailedUser>> = observable(Map());
  #usersByKey: WritableObservable<Map<string, UsersPagination>> = observable(Map());
  #currentUserId: WritableObservable<Loadable<number>> = observable(NotLoaded);

  public getUser = (id: number): Observable<Loadable<DetailedUser>> => {
    return this.#users.select((map) => {
      const user = map.get(id);
      return user ? Loaded(user) : NotLoaded;
    });
  };

  public getCurrentUser = (): Observable<Loadable<DetailedUser>> => {
    return this.#users.select((map) => {
      const id = this.#currentUserId.get();
      // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
      return Loadable.map(id, (id) => map.get(id)!);
    });
  };

  public ensureCurrentUserFetched = (canceler: AbortController, hard = false) => {
    if (!hard && this.#currentUserId.get() !== NotLoaded) return;

    getCurrentUser({ signal: canceler.signal })
      .then((response) => {
        this.updateUsers([response]);
        this.#currentUserId.set(Loaded(response.id));
      })
      .catch((e) => handleError(e, { publicSubject: 'Unable to fetch current user.' }));
  };

  public updateCurrentUser = (id: number | null) => {
    if (id === null) this.#currentUserId.set(NotLoaded);
    else this.#currentUserId.set(Loaded(id));
  };

  public getUsers = (cfg?: FetchUsersConfig): Observable<Loadable<DetailedUserList>> => {
    const config = cfg ?? {};

    return this.#usersByKey.select((map) => {
      const usersPagination = map.get(encodeParams(config));

      if (!usersPagination) return NotLoaded;

      const userPage: DetailedUserList = {
        pagination: usersPagination.pagination,
        users: usersPagination.users.flatMap((userId) => {
          const user = this.#users.get().get(userId);

          return user ? [user] : [];
        }),
      };

      return Loaded(userPage);
    });
  };

  public ensureUsersFetched = (canceler: AbortController, cfg?: FetchUsersConfig, hard = false) => {
    const config = cfg ?? {};
    const usersPagination = this.#usersByKey.get().get(encodeParams(config));

    if (!hard && usersPagination) return;

    getUsers(config, { signal: canceler?.signal })
      .then((response) => {
        this.updateUsersByKey(config, response);
        this.updateUsers(response.users);
      })
      .catch((e) => handleError(e, { publicSubject: 'Unable to fetch users.' }));
  };

  public updateUsers = (users: DetailedUser | DetailedUser[]) => {
    this.#users.update((map) => {
      return map.withMutations((map) => {
        if (Array.isArray(users)) users.forEach((user) => map.set(user.id, user));
        else map.set(users.id, users);
      });
    });
  };

  private updateUsersByKey = (
    config: FetchUsersConfig | Record<string, never>,
    usersList: DetailedUserList,
  ) => {
    const usersPages = {
      pagination: usersList.pagination,
      users: usersList.users.map((user) => user.id),
    };

    this.#usersByKey.update((map) => map.set(encodeParams(config), usersPages));
  };
}

const usersStore = new UsersService();

export { FetchUsersConfig };

export default usersStore;
