import { useCallback } from 'react';

import { generateContext } from 'contexts';
import { getInfo } from 'services/api';
import { DeterminedInfo } from 'types';
import { isEqual } from 'utils/data';

const Info = generateContext<DeterminedInfo>({
  initialState: {
    clusterId: '',
    clusterName: '',
    isTelemetryEnabled: false,
    masterId: '',
    version: process.env.VERSION || '',
  },
  name: 'DeterminedInfo',
});

export const useFetchInfo = (canceler: AbortController): () => Promise<void> => {
  const info = Info.useStateContext();
  const setInfo = Info.useActionContext();

  return useCallback(async (): Promise<void> => {
    try {
      const infoResponse = await getInfo({ signal: canceler.signal });

      if (!isEqual(info, infoResponse)) {
        setInfo({ type: Info.ActionType.Set, value: infoResponse });
      }
    } catch (e) {}
  }, [ canceler, info, setInfo ]);
};

export default Info;
