import { useCallback, useState } from 'react';

import usePolling from 'hooks/usePolling';
import { queryTrials } from 'services/api';
import handleError from 'utils/error';

import { encodeFilters, encodeTrialSorter } from '../api';
import { TrialFilters, TrialSorter } from '../Collections/filters';

import { decodeTrialsWithMetadata, defaultTrialData, TrialsWithMetadata } from './data';

interface Params {
  filters: TrialFilters;
  limit: number;
  offset: number;
  sorter: TrialSorter;
}
export const useFetchTrials = ({
  filters,
  limit,
  offset,
  sorter,
}: Params): TrialsWithMetadata => {
  const [ trials, setTrials ] = useState<TrialsWithMetadata>(defaultTrialData());
  const fetchTrials = useCallback(async () => {
    let response: any;
    const _filters = encodeFilters(filters);
    const _sorter = encodeTrialSorter(sorter);
    try {
      response = await queryTrials({
        filters: _filters,
        limit,
        offset,
        sorter: _sorter,
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch trials.' });
    }
    const newTrials = decodeTrialsWithMetadata(response.trials);
    if (newTrials)
      setTrials(newTrials);

  }, [ filters, limit, offset, sorter ]);

  usePolling(fetchTrials, { interval: 200000, rerunOnNewFn: true });

  return trials;
};
