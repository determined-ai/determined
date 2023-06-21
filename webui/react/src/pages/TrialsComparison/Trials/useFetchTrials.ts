import { useCallback, useEffect, useState } from 'react';

import usePolling from 'hooks/usePolling';
import { queryTrials } from 'services/api';
import { V1QueryTrialsResponse } from 'services/api-ts-sdk';
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
}: Params): { loading: boolean; refetch: () => void; trials: TrialsWithMetadata } => {
  const [trials, setTrials] = useState<TrialsWithMetadata>(defaultTrialData());
  const [loading, setLoading] = useState(false);
  const fetchTrials = useCallback(async () => {
    let response: V1QueryTrialsResponse | undefined = undefined;
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
    } finally {
      setLoading(false);
    }
    const newTrials = decodeTrialsWithMetadata(response);
    setTrials({ ...newTrials, ids: newTrials.ids.slice(0, limit) });
  }, [filters, limit, offset, sorter]);

  useEffect(() => {
    setLoading(true);
    fetchTrials();
  }, [fetchTrials]);

  usePolling(fetchTrials, { interval: 10000 });

  return { loading, refetch: fetchTrials, trials };
};
