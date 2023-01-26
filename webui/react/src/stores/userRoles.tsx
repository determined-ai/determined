import { observable, WritableObservable } from 'micro-observables';
import React, { createContext, PropsWithChildren, useCallback, useContext, useRef } from 'react';

import { getPermissionsSummary } from 'services/api';
import { isEqual } from 'shared/utils/data';
import { UserAssignment, UserRole } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { useValueMemoizedObservable } from 'utils/observable';

type UserRolesAndAssignmentsContext = {
  #userAssignments: WritableObservable<Loadable<UserAssignment[]>>;
  #userRoles: WritableObservable<Loadable<UserRole[]>>;
};

const UserRolesAndAssignmentsContext = createContext<UserRolesAndAssignmentsContext | null>(null);

export const UserRolesProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const userRoles = useRef<WritableObservable<Loadable<UserRole[]>>>(observable(NotLoaded));
  const userAssignments = useRef<WritableObservable<Loadable<UserAssignment[]>>>(
    observable(NotLoaded),
  );

  return (
    <UserRolesAndAssignmentsContext.Provider
      value={{
        userAssignments: userAssignments.current,
        userRoles: userRoles.current,
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

  const { userAssignments, userRoles } = context;

  return useCallback(async (): Promise<void> => {
    try {
      const { assignments, roles } = await getPermissionsSummary({ signal: canceler.signal });
      userAssignments.set(Loaded(assignments));
      userRoles.set(Loaded(roles));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, userAssignments, userRoles]);
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
  const { userAssignments, userRoles } = context;
  const memoAssignments = useValueMemoizedObservable(userAssignments);
  const memoRoles = useValueMemoizedObservable(userRoles);

  return useCallback(async (): Promise<void> => {
    if (memoAssignments !== NotLoaded && memoRoles !== NotLoaded) return;
    try {
      const { roles, assignments } = await getPermissionsSummary({ signal: canceler.signal });
      if (!isEqual(memoAssignments, assignments)) userAssignments.set(Loaded(assignments));
      if (!isEqual(memoRoles, roles)) userRoles.set(Loaded(roles));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, memoRoles, userRoles, memoAssignments, userAssignments]);
};

export const useUserRoles = (): Loadable<UserRole[]> => {
  const context = useContext(UserRolesAndAssignmentsContext);
  if (context === null) {
    throw new Error('Attempted to use useUserRoles outside of UserRolesAndAssignments Context');
  }

  const { userRoles } = context;
  const userRoleState = useValueMemoizedObservable(userRoles);
  return userRoleState;
};

export const useUserAssignments = (): Loadable<UserAssignment[]> => {
  const context = useContext(UserRolesAndAssignmentsContext);
  if (context === null) {
    throw new Error(
      'Attempted to use useUserAssignments outside of UserRolesAndAssignments Context',
    );
  }
  const { userAssignments } = context;
  const userAssignmentState = useValueMemoizedObservable(userAssignments);
  return userAssignmentState;
};

export const useResetUserAssignmentsAndRoles = (): (() => void) => {
  const context = useContext(UserRolesAndAssignmentsContext);
  if (context === null) {
    throw new Error('Attempted to use logoutUserRoles outside of UserRolesAndAssignmentsContext');
  }
  const { userAssignments, userRoles } = context;

  return useCallback((): void => {
    userAssignments.set(NotLoaded);
    userRoles.set(NotLoaded);
  }, [userAssignments, userRoles]);
};
