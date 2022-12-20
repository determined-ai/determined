import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getCurrentUser, getUsers } from 'services/api';
import { V1GetUsersRequestSortBy } from 'services/api-ts-sdk';
import { isEqual } from 'shared/utils/data';
import { DetailedUser, DetailedUserList } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type UsersContext = {
  currentUser: Loadable<DetailedUser>;
  updateCurrentUser: (fn: (currentUser: Loadable<DetailedUser>) => Loadable<DetailedUser>) => void;
  updateUsers: (fn: (users: Loadable<DetailedUser[]>) => Loadable<DetailedUser[]>) => void;
  users: Loadable<DetailedUser[]>;
};

type UseCurentUserReturn = {
  currentUser: Loadable<DetailedUser>;
  updateCurrentUser: (user: DetailedUser, users?: DetailedUser[]) => void;
};

type FetchUsersConfig = {
  limit: number;
  offset: number;
  orderBy: 'ORDER_BY_DESC' | 'ORDER_BY_ASC';
  sortBy: V1GetUsersRequestSortBy;
};

const UsersContext = createContext<UsersContext | null>(null);

export const UsersProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [users, setUsers] = useState<Loadable<DetailedUser[]>>(NotLoaded);
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

export const useFetchUsers = (
  canceler: AbortController,
): ((cfg?: FetchUsersConfig) => Promise<DetailedUserList | void>) => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useFetchUsers outside of Users Context');
  }

  const { updateUsers } = context;

  return useCallback(
    async (cfg?: FetchUsersConfig): Promise<DetailedUserList> => {
      const config = cfg ?? {};
      const response = await getUsers(config, { signal: canceler.signal });

      updateUsers(() => Loaded(response.users));

      return response;
    },
    [canceler, updateUsers],
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
      if (users !== NotLoaded) return;

      try {
        const config = cfg ?? {};
        const response = await getUsers(config, { signal: canceler.signal });

        updateUsers(() => Loaded(response.users));
      } catch (e) {
        handleError(e);
      }
    },
    [canceler, updateUsers, users],
  );
};

export const useUsers = (): Loadable<DetailedUser[]> => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useUsers outside of Users Context');
  }

  return context.users;
};

export const useCurrentUsers = (): UseCurentUserReturn => {
  const context = useContext(UsersContext);

  if (context === null) {
    throw new Error('Attempted to use useCurrentUser outside of User Context');
  }
  const { currentUser, updateCurrentUser, updateUsers } = context;

  const userUpdateCallback = useCallback(
    (user: DetailedUser, users: DetailedUser[] = []) => {
      const usersArray = [...users];

      updateCurrentUser(() => {
        const userIdx = usersArray.findIndex((changeUser) => changeUser.id === user.id);

        if (userIdx > -1) usersArray[userIdx] = { ...usersArray[userIdx], ...user };

        return Loaded(user);
      });

      updateUsers((prevState) => {
        if (!Loadable.isLoaded(prevState) || !usersArray.length) return prevState;

        if (isEqual(prevState.data, usersArray)) return prevState;

        return Loaded(usersArray);
      });
    },
    [updateCurrentUser, updateUsers],
  );

  return {
    currentUser,
    updateCurrentUser: userUpdateCallback,
  };
};
