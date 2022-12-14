import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { listRoles } from 'services/api';
import { noOp } from 'shared/utils/service';
import { UserRole } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type KnownRolesContext = {
  knownRoles: Loadable<UserRole[]>;
  updateKnowRoles: React.Dispatch<React.SetStateAction<Loadable<UserRole[]>>>;
};

export const initKnowRoles: UserRole[] = [];

const KnownRolesContext = createContext<KnownRolesContext>({
  knownRoles: NotLoaded,
  updateKnowRoles: noOp,
});

export const KnownRolesProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [knownRoles, updateKnowRoles] = useState<Loadable<UserRole[]>>(NotLoaded);
  return (
    <KnownRolesContext.Provider value={{ knownRoles, updateKnowRoles }}>
      {children}
    </KnownRolesContext.Provider>
  );
};

export const useFetchKnownRoles = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(KnownRolesContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchKnownRoles outside of KnownRoles Context');
  }
  const { updateKnowRoles } = context;
  return useCallback(async (): Promise<void> => {
    try {
      const response = await listRoles({ limit: 0 }, { signal: canceler.signal });
      updateKnowRoles(Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateKnowRoles]);
};

export const useEnsureKnownRolesFetched = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(KnownRolesContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchKnownRoles outside of KnownRoles Context');
  }
  const { knownRoles, updateKnowRoles } = context;
  return useCallback(async (): Promise<void> => {
    if (knownRoles !== NotLoaded) return;
    try {
      const response = await listRoles({ limit: 0 }, { signal: canceler.signal });
      updateKnowRoles(Loaded(response));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, knownRoles, updateKnowRoles]);
};

export const useKnownRoles = (): Loadable<UserRole[]> => {
  const context = useContext(KnownRolesContext);

  if (context === null) {
    throw new Error('Attempted to use useFetchKnownRoles outside of KnownRoles Context');
  }
  return context.knownRoles;
};
