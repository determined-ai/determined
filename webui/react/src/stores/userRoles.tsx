import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getPermissionsSummary } from 'services/api';
import { UserAssignment, UserRole } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type UserRolesAndAssignmentsContext = {
  updateUserAssignments: (
    fn: (r: Loadable<UserAssignment[]>) => Loadable<UserAssignment[]>,
  ) => void;
  updateUserRoles: (fn: (r: Loadable<UserRole[]>) => Loadable<UserRole[]>) => void;
  userAssignments: Loadable<UserAssignment[]>;
  userRoles: Loadable<UserRole[]>;
};

const UserRolesAndAssignmentsContext = createContext<UserRolesAndAssignmentsContext | null>(null);

export const UserRolesProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [userRoles, setUserRoles] = useState<Loadable<UserRole[]>>(NotLoaded);
  const [userAssignments, setUserAssignments] = useState<Loadable<UserAssignment[]>>(NotLoaded);
  return (
    <UserRolesAndAssignmentsContext.Provider
      value={{
        updateUserAssignments: setUserAssignments,
        updateUserRoles: setUserRoles,
        userAssignments: userAssignments,
        userRoles: userRoles,
      }}>
      {children}
    </UserRolesAndAssignmentsContext.Provider>
  );
};

export const useFetchUserRolesAndAssignments = (
  canceler: AbortController,
): (() => Promise<void>) => {
  const context = useContext(UserRolesAndAssignmentsContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchUserRoles outside of UserRolesAndAssignmentsContext');
  }
  const { updateUserRoles, updateUserAssignments } = context;

  return useCallback(async (): Promise<void> => {
    try {
      const { roles, assignments } = await getPermissionsSummary({ signal: canceler.signal });
      updateUserRoles(() => Loaded(roles));
      updateUserAssignments(() => Loaded(assignments));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateUserRoles, updateUserAssignments]);
};

export const useEnsureUserRolesAndAssignmentsFetched = (
  canceler: AbortController,
): (() => Promise<void>) => {
  const context = useContext(UserRolesAndAssignmentsContext);
  if (context === null) {
    throw new Error(
      'Attempted to use useEnsureFetchUserRoles outside of UserRolesAndAssignments Context',
    );
  }
  const { userRoles, updateUserRoles, userAssignments, updateUserAssignments } = context;

  return useCallback(async (): Promise<void> => {
    if (userRoles !== NotLoaded && userAssignments !== NotLoaded) return;
    try {
      const { roles, assignments } = await getPermissionsSummary({ signal: canceler.signal });
      updateUserRoles(() => Loaded(roles));
      updateUserAssignments(() => Loaded(assignments));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, userRoles, updateUserRoles, userAssignments, updateUserAssignments]);
};

export const useUserRoles = (): Loadable<UserRole[]> => {
  const context = useContext(UserRolesAndAssignmentsContext);
  if (context === null) {
    throw new Error('Attempted to use useUserRoles outside of UserRolesAndAssignments Context');
  }

  const { userRoles } = context;

  return userRoles;
};

export const useUserAssignments = (): Loadable<UserAssignment[]> => {
  const context = useContext(UserRolesAndAssignmentsContext);
  if (context === null) {
    throw new Error(
      'Attempted to use useUserAssignments outside of UserRolesAndAssignments Context',
    );
  }
  const { userAssignments } = context;

  return userAssignments;
};
