// currentUser: Loadable<number>;
// updateCurrentUser: (fn: (currentUser: Loadable<number>) => Loadable<number>) => void;
// updateUsers: (fn: (users: Map<number, DetailedUser>) => Map<number, DetailedUser>) => void;
// updateUsersByKey: (
//   fn: (users: Map<string, UsersPagination>) => Map<string, UsersPagination>,
// ) => void;
// users: Map<number, DetailedUser>;
// usersByKey: Map<string, UsersPagination>;

import { Map } from 'immutable';
import { observable, WritableObservable } from 'micro-observables';

import { getCurrentUser, getUsers } from 'services/api';
import { V1Pagination } from 'services/api-ts-sdk';
import { DetailedUser, DetailedUserList } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { encodeParams } from 'utils/store';

import { FetchUsersConfig } from './users';

type UsersPagination = {
  pagination: V1Pagination;
  users: number[];
};

class UsersService {
  #users: WritableObservable<Map<number, DetailedUser>> = observable(Map());
  #usersByKey: WritableObservable<Map<string, UsersPagination>> = observable(Map());
  #currentUserId: WritableObservable<Loadable<number>> = observable(NotLoaded);

  public getCurrentUser = () => {
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    return Loadable.map(this.#currentUserId.get(), (userId) => this.#users.get().get(userId)!);
  };

  public ensureCurrentUserFetched = async (canceler: AbortController): Promise<void> => {
    if (this.#currentUserId.get() !== NotLoaded) return;

    try {
      const response = await getCurrentUser({ signal: canceler.signal });

      this.updateUsers([response]);
      this.#currentUserId.set(Loaded(response.id));
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch current user.' });
    }
  };

  public updateCurrentUser = (id: number) => {
    this.#currentUserId.set(Loaded(id));
  };

  public ensureUsersFetched = async (
    cfg?: FetchUsersConfig,
    canceler?: AbortController,
  ): Promise<void> => {
    const config = cfg ?? {};
    const usersPagination = this.#usersByKey.get().get(encodeParams(config));

    if (usersPagination) return;

    try {
      const response = await getUsers(config, { signal: canceler?.signal });

      this.updateUsersByKey(config, response);
      this.updateUsers(response.users);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch users.' });
    }
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

  private updateUsers = (users: DetailedUser[]) => {
    this.#users.update((map) => {
      return map.withMutations((map) => {
        users.forEach((user) => map.set(user.id, user));
      });
    });
  };
}

const usersStore = new UsersService();

export default usersStore;
