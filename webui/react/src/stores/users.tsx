import { Map } from 'immutable';
import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getCurrentUser, getUsers } from 'services/api';
import { V1GetUsersRequestSortBy, V1Pagination } from 'services/api-ts-sdk';
import { DetailedUser } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { encodeParams } from 'utils/store';

type UsersPagination = {
  pagination: V1Pagination;
  users: number[];
};

export type UserPage = {
  pagination: V1Pagination;
  users: DetailedUser[];
};

type UsersContext = {
  currentUser: Loadable<number>;
  updateCurrentUser: (fn: (currentUser: Loadable<number>) => Loadable<number>) => void;
  updateUsers: (fn: (users: Map<number, DetailedUser>) => Map<number, DetailedUser>) => void;
  updateUsersByKey: (
    fn: (users: Map<string, UsersPagination>) => Map<string, UsersPagination>,
  ) => void;
  users: Map<number, DetailedUser>;
  usersByKey: Map<string, UsersPagination>;
};

export type FetchUsersConfig = {
  limit: number;
  offset: number;
  orderBy: 'ORDER_BY_DESC' | 'ORDER_BY_ASC';
  sortBy: V1GetUsersRequestSortBy;
};

const UsersContext = createContext<UsersContext | null>(null);

export const UsersProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [usersByKey, setUsersByKey] = useState<Map<string, UsersPagination>>(
    Map<string, UsersPagination>(),
  );
  const [users, setUsers] = useState<Map<number, DetailedUser>>(Map<number, DetailedUser>());
  const [currentUser, setCurrentUser] = useState<Loadable<number>>(NotLoaded);

  return (
    <UsersContext.Provider
      value={{
        currentUser,
        updateCurrentUser: setCurrentUser,
        updateUsers: setUsers,
        updateUsersByKey: setUsersByKey,
        users,
        usersByKey,
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

  const { updateUsersByKey, updateUsers } = context;

  return useCallback(
    async (cfg?: FetchUsersConfig) => {
      try {
        const config = cfg ?? {};
        const response = await getUsers(config, { signal: canceler.signal });
        const usersPages = {
          pagination: response.pagination,
          users: response.users.map((user) => user.id),
        };

        updateUsersByKey((prevState) => prevState.set(encodeParams(config), usersPages));
        updateUsers((prevState) => {
          return prevState.withMutations((map) => {
            response.users.forEach((user) => map.set(user.id, user));
          });
        });
      } catch (e) {
        handleError(e, { publicSubject: 'Unable to fetch users.' });
      }
    },
    [canceler, updateUsers, updateUsersByKey],
  );
};

export const useEnsureCurrentUserFetched = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useEnsureCurrentUserFetched outside of Users Context');
  }

  const { updateCurrentUser, currentUser, updateUsers } = context;

  return useCallback(async (): Promise<void> => {
    if (currentUser !== NotLoaded) return;

    try {
      const response = await getCurrentUser({ signal: canceler.signal });

      updateUsers((prevState) => prevState.set(response.id, response));
      updateCurrentUser(() => Loaded(response.id));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateCurrentUser, currentUser, updateUsers]);
};

export const useEnsureUsersFetched = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useFetchUsers outside of Users Context');
  }

  const { updateUsers, updateUsersByKey, usersByKey } = context;

  return useCallback(
    async (cfg?: FetchUsersConfig): Promise<void> => {
      const config = cfg ?? {};
      const usersPagination = usersByKey.get(encodeParams(config));

      if (usersPagination) return;

      try {
        const response = await getUsers(config, { signal: canceler.signal });
        const usersPages = {
          pagination: response.pagination,
          users: response.users.map((user) => user.id),
        };

        updateUsersByKey((prevState) => prevState.set(encodeParams(config), usersPages));
        updateUsers((prevState) => {
          return prevState.withMutations((map) => {
            response.users.forEach((user) => map.set(user.id, user));
          });
        });
      } catch (e) {
        handleError(e);
      }
    },
    [canceler, updateUsers, usersByKey, updateUsersByKey],
  );
};

export const useUsers = (cfg?: FetchUsersConfig): Loadable<UserPage> => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useUsers outside of Users Context');
  }
  const config = cfg ?? {};
  const usersPagination = context.usersByKey.get(encodeParams(config));

  if (!usersPagination) return NotLoaded;

  const userPage = {
    pagination: usersPagination.pagination,
    users: usersPagination.users.flatMap((userId) => {
      const user = context.users.get(userId);

      return user ? [user] : [];
    }),
  };

  return Loaded(userPage);
};

export const useUpdateCurrentUser = (): ((id: number) => void) => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useUpdateCurrentUser outside of Users Context');
  }

  const { updateCurrentUser } = context;
  const callback = useCallback(
    (id: number) => {
      updateCurrentUser(() => Loaded(id));
    },
    [updateCurrentUser],
  );

  return callback;
};

export const useUpdateUser = (): ((
  id: number,
  updater: (arg0: DetailedUser) => DetailedUser,
) => void) => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useUpdateUser outside of Users Context');
  }

  const { updateUsers } = context;
  const callback = useCallback(
    (id: number, updater: (arg0: DetailedUser) => DetailedUser) => {
      updateUsers((prevState) => {
        if (prevState.has(id)) {
          // this state is statically guaranteed to be non-null
          // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
          return prevState.update(id, (detailedUser) => updater(detailedUser!));
        }

        return prevState;
      });
    },
    [updateUsers],
  );

  return callback;
};

export const useCurrentUser = (): Loadable<DetailedUser> => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useCurrentUser outside of User Context');
  }
  const { currentUser, users } = context; // this state is statically guaranteed to be non-null

  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
  const loadedUser = Loadable.map(currentUser, (userId) => users.get(userId)!);

  return loadedUser;
};
