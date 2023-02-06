import { useCallback } from 'react';

import { getInfo } from 'services/api';
import { DeterminedInfo } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { observable, useObservable } from 'utils/observable';

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

const info = observable<Loadable<DeterminedInfo>>(NotLoaded);

export const useFetchDeterminedInfo = (canceler: AbortController) => {
  return useCallback(async () => {
    try {
      const response = await getInfo({ signal: canceler.signal });
      info.set(Loaded(response));
    } catch (e) {
      info.update((prevInfo) => {
        const info = Loadable.getOrElse(initInfo, prevInfo);
        return Loaded({ ...info, checked: true });
      });
      handleError(e);
    }
  }, [canceler]);
};

export const useDeterminedInfo = () => {
  return useObservable(info.readOnly());
};
