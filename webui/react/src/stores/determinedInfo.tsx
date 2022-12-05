import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getInfo } from 'services/api';
import { noOp } from 'shared/utils/service';
import { DeterminedInfo } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type DeterminedInfoContext = {
  info: Loadable<DeterminedInfo>;
  updateInfo: (a: Loadable<DeterminedInfo>) => void;
};

export const initInfo: DeterminedInfo = {
  branding: undefined,
  checked: false,
  clusterId: '',
  clusterName: '',
  featureSwitches: [],
  isTelemetryEnabled: false,
  masterId: '',
  rbacEnabled: false,
  version: process.env.VERSION || '',
};

const DeterminedInfoContext = createContext<DeterminedInfoContext>({
  info: NotLoaded,
  updateInfo: noOp,
});

export const DeterminedInfoProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [info, setInfo] = useState<Loadable<DeterminedInfo>>(NotLoaded);
  return (
    <DeterminedInfoContext.Provider value={{ info: info, updateInfo: setInfo }}>
      {children}
    </DeterminedInfoContext.Provider>
  );
};

export const useFetchInfo = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(DeterminedInfoContext);
  if (context === null) {
    throw new Error('Attempted to use useDeterminedInfo outside of Determinednfo Context');
  }
  const { updateInfo, info } = context;
  return useCallback(async (): Promise<void> => {
    try {
      const response = await getInfo({ signal: canceler.signal });
      updateInfo(Loaded(response));
    } catch (e) {
      updateInfo(Loaded({ ...Loadable.getOrElse(initInfo, info), checked: true }));
      handleError(e);
    }
  }, [canceler, updateInfo, info]);
};

export const useEnsureInfoFetched = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(DeterminedInfoContext);
  if (context === null) {
    throw new Error('Attempted to use useDeterminedInfo outside of Determinednfo Context');
  }
  const { info, updateInfo } = context;
  return useCallback(async (): Promise<void> => {
    if (info !== NotLoaded) return;
    try {
      const response = await getInfo({ signal: canceler.signal });
      updateInfo(Loaded(response));
    } catch (e) {
      updateInfo(Loaded({ ...Loadable.getOrElse(initInfo, info), checked: true }));
      handleError(e);
    }
  }, [canceler, updateInfo, info]);
};

export const useDeterminedInfo = (): Loadable<DeterminedInfo> => {
  const context = useContext(DeterminedInfoContext);

  if (context === null) {
    throw new Error('Attempted to use useDeterminedInfo outside of Determinednfo Context');
  }
  return context.info;
};
