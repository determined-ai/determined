import { useEffect, useState } from 'react';

import useUI from 'components/kit/Theme';
import { V1GetTrialProfilerAvailableSeriesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';

import { AvailableSeries } from './types';

export const useFetchProfilerSeries = (trialId: number): AvailableSeries => {
  const { ui } = useUI();
  const [availableSeries, setAvailableSeries] = useState<AvailableSeries>({});

  useEffect(() => {
    if (ui.isPageHidden) return;

    const canceler = new AbortController();

    readStream(
      detApi.StreamingProfiler.getTrialProfilerAvailableSeries(trialId, true, {
        signal: canceler.signal,
      }),
      (event: V1GetTrialProfilerAvailableSeriesResponse) => {
        const newAvailableSeries: AvailableSeries = {};

        event.labels.forEach((label) => {
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
  }, [trialId, ui.isPageHidden]);

  return availableSeries;
};
