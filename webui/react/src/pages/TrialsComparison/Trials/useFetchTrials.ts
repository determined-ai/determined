import { useCallback, useEffect, useState } from 'react';

import { queryTrials } from 'services/api';
import { V1AugmentedTrial } from 'services/api-ts-sdk';
import { clone } from 'shared/utils/data';
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
  const [ trials, setTrials ] = useState<TrialsWithMetadata>(clone(defaultTrialData));
  const fetchTrials = useCallback(async () => {
    let trials: V1AugmentedTrial[] = [];
    const _filters = encodeFilters(filters);
    const _sorter = encodeTrialSorter(sorter);
    try {
      trials = await queryTrials({
        filters: _filters,
        limit,
        offset,
        sorter: _sorter,
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch trials.' });
    }

    const newTrials = decodeTrialsWithMetadata(trials);
    setTrials(newTrials);

  }, [ filters, limit, offset, sorter ]);

  useEffect(() => { fetchTrials(); }, [ fetchTrials ]);

  // usePolling(fetchTrials, { interval: 10000, rerunOnNewFn: true });

  return trials;
};
