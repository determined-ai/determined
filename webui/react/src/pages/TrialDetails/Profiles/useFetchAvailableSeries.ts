import { useEffect, useState } from 'react';

import { V1GetTrialProfilerAvailableSeriesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';

import { AvailableSeries } from './types';

export const useFetchAvailableSeries = (trialId: number): AvailableSeries => {
  const [ availableSeries, setAvailableSeries ] = useState<AvailableSeries>({});

  useEffect(() => {
    const canceler = new AbortController();

    consumeStream(
      detApi.StreamingProfiler.determinedGetTrialProfilerAvailableSeries(
        trialId,
        true,
        { signal: canceler.signal },
      ),
      (event: V1GetTrialProfilerAvailableSeriesResponse) => {
        const newAvailableSeries: AvailableSeries = {};

        event.labels.forEach(label => {
          const agentId: string = label.agentId as unknown as string;
          // The reported gpuuuid can be empty since the slot might not be backed by gpu.
          const gpuUuid: string = label.gpuUuid as unknown as string;
          const metricType: string = label.metricType as unknown as string;
          const name: string = label.name as unknown as string;

          if (!newAvailableSeries[metricType]) {
            newAvailableSeries[metricType] = {};
          }
          if (!newAvailableSeries[metricType][name]) {
            newAvailableSeries[metricType][name] = {};
          }
          if (!newAvailableSeries[metricType][name][agentId]) {
            newAvailableSeries[metricType][name][agentId] = [];
          }
          newAvailableSeries[metricType][name][agentId].push(gpuUuid);
        });

        setAvailableSeries(newAvailableSeries);
      },
    );

    return () => canceler.abort();
  }, [ trialId ]);

  return availableSeries;
};
