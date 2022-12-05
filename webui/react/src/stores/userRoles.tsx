import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getUserRoles } from 'services/api';
import { UserRole } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type UserRolesContext = {
  updateUserRoles: (fn: (r: Loadable<UserRole[]>) => Loadable<UserRole[]>) => void;
  userRoles: Loadable<UserRole[]>;
};

const UserRolesContext = createContext<UserRolesContext | null>(null);

export const UserRolesProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [state, setState] = useState<Loadable<UserRole[]>>(NotLoaded);
  return (
    <UserRolesContext.Provider value={{ updateUserRoles: setState, userRoles: state }}>
      {children}
    </UserRolesContext.Provider>
  );
};

export const useFetchUserRoles = (
  canceler: AbortController,
  userId?: number,
): (() => Promise<void>) => {
  const context = useContext(UserRolesContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchUserRoles outside of UserRoles Context');
  }
  const { updateUserRoles } = context;

  return useCallback(async (): Promise<void> => {
    if (!userId) return;
    try {
      const response = await getUserRoles({ userId }, { signal: canceler.signal });
      updateUserRoles(() => Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateUserRoles, userId]);
};

export const useEnsureUserRolesFetched = (
  canceler: AbortController,
  userId?: number,
): (() => Promise<void>) => {
  const context = useContext(UserRolesContext);
  if (context === null) {
    throw new Error('Attempted to use useEnsureFetchUserRoles outside of UserRoles Context');
  }
  const { userRoles, updateUserRoles } = context;

  return useCallback(async (): Promise<void> => {
    if (userRoles !== NotLoaded) return;
    if (!userId) return;
    try {
      const response = await getUserRoles({ userId }, { signal: canceler.signal });
      updateUserRoles(() => Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, userId, userRoles, updateUserRoles]);
};

export const useUserRoles = (): Loadable<UserRole[]> => {
  const context = useContext(UserRolesContext);
  if (context === null) {
    throw new Error('Attempted to use useUserRoles outside of UserRoles Context');
  }
  const { userRoles } = context;

  return userRoles;
};
