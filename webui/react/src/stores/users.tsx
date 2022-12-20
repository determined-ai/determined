import { Map } from 'immutable';
import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getCurrentUser, getUsers } from 'services/api';
import { V1GetUsersRequestSortBy, V1Pagination } from 'services/api-ts-sdk';
import { isEqual } from 'shared/utils/data';
import { DetailedUser } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { encodeParams } from 'utils/store';

type UsersPagination = {
  pagination: V1Pagination;
  users: DetailedUser[];
};

type UsersContext = {
  currentUser: Loadable<DetailedUser>;
  updateCurrentUser: (fn: (currentUser: Loadable<DetailedUser>) => Loadable<DetailedUser>) => void;
  updateUsers: (users: Map<string, UsersPagination>) => void;
  users: Map<string, UsersPagination>;
};

type UseCurentUserReturn = {
  currentUser: Loadable<DetailedUser>;
  updateCurrentUser: (user: DetailedUser, users?: DetailedUser[]) => void;
};

export type FetchUsersConfig = {
  limit: number;
  offset: number;
  orderBy: 'ORDER_BY_DESC' | 'ORDER_BY_ASC';
  sortBy: V1GetUsersRequestSortBy;
};

const UsersContext = createContext<UsersContext | null>(null);

export const UsersProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [users, setUsers] = useState<Map<string, UsersPagination>>(() =>
    Map<string, UsersPagination>(),
  );
  const [currentUser, setCurrentUser] = useState<Loadable<DetailedUser>>(NotLoaded);

  return (
    <UsersContext.Provider
      value={{
        currentUser,
        updateCurrentUser: setCurrentUser,
        updateUsers: setUsers,
        users,
      }}>
      {children}
    </UsersContext.Provider>
  );
};

export const useFetchUsers = (canceler: AbortController): ((cfg?: FetchUsersConfig) => void) => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useFetchUsers outside of Users Context');
  }

  const { users, updateUsers } = context;

  return useCallback(
    async (cfg?: FetchUsersConfig) => {
      try {
        const config = cfg ?? {};
        const response = await getUsers(config, { signal: canceler.signal });

        updateUsers(users.set(encodeParams(config), response));
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to fetch users.' });
      }
    },
    [canceler, updateUsers, users],
  );
};

export const useEnsureCurrentUserFetched = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useEnsureCurrentUserFetched outside of Users Context');
  }

  const { updateCurrentUser, currentUser } = context;

  return useCallback(async (): Promise<void> => {
    if (currentUser !== NotLoaded) return;

    try {
      const response = await getCurrentUser({ signal: canceler.signal });

      updateCurrentUser(() => Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateCurrentUser, currentUser]);
};

export const useEnsureUsersFetched = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useFetchUsers outside of Users Context');
  }

  const { updateUsers, users } = context;

  return useCallback(
    async (cfg?: FetchUsersConfig): Promise<void> => {
      const config = cfg ?? {};
      const usersPagination = users.get(encodeParams(config));

      if (usersPagination) return;

      try {
        const response = await getUsers(config, { signal: canceler.signal });

        updateUsers(users.set(encodeParams(config), response));
      } catch (e) {
        handleError(e);
      }
    },
    [canceler, updateUsers, users],
  );
};

export const useUsers = (cfg?: FetchUsersConfig): Loadable<DetailedUser[]> => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useUsers outside of Users Context');
  }
  const config = cfg ?? {};
  const usersPagination = context.users.get(encodeParams(config));

  return usersPagination ? Loaded(usersPagination.users) : NotLoaded;
};

export const useUsersPagination = (cfg?: FetchUsersConfig): Loadable<V1Pagination> => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useUsersPagination outside of Users Context');
  }

  const config = cfg ?? {};
  const usersPagination = context.users.get(encodeParams(config));

  return usersPagination ? Loaded(usersPagination.pagination) : NotLoaded;
};

export const useCurrentUsers = (): UseCurentUserReturn => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useCurrentUser outside of User Context');
  }
  const { currentUser, users: usersPagination, updateCurrentUser, updateUsers } = context;

  const userUpdateCallback = useCallback(
    (user: DetailedUser, users: DetailedUser[] = []) => {
      const usersArray = [...users];

      updateCurrentUser(() => {
        const userIdx = usersArray.findIndex((changeUser) => changeUser.id === user.id);

        if (userIdx > -1) usersArray[userIdx] = { ...usersArray[userIdx], ...user };

        return Loaded(user);
      });

      const cachedUsers = usersPagination.get(encodeParams({}));

      if (cachedUsers && usersArray.length && !isEqual(cachedUsers.users, usersArray)) {
        updateUsers(
          usersPagination.set(encodeParams({}), {
            pagination: cachedUsers.pagination,
            users: usersArray,
          }),
        );
      }
    },
    [usersPagination, updateCurrentUser, updateUsers],
  );

  return {
    currentUser,
    updateCurrentUser: userUpdateCallback,
  };
};
