import { useEffect, useState } from 'react';

import { terminalRunStates } from 'constants/states';
import { useStore } from 'contexts/Store';
import { V1GetTrialProfilerMetricsResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { RunState } from 'types';
import { clone, hasObjectKeys } from 'utils/data';

import { MetricsAggregateInterface, MetricType } from './types';

const DEFAULT_DATA: MetricsAggregateInterface = {
  dataByTime: {},
  isEmpty: true,
  isLoading: true,
  names: [ 'batch' ],
};

export const useFetchMetrics = (
  trialId: number,
  trialState: RunState,
  labelsMetricType: MetricType,
  labelsName: string|undefined = undefined,
  labelsAgentId: string|undefined = undefined,
  labelsGpuUuid: string|undefined = undefined,
): MetricsAggregateInterface => {
  const { ui } = useStore();
  const [ data, setData ] = useState<MetricsAggregateInterface>(clone(DEFAULT_DATA));

  useEffect(() => {
    if (ui.isPageHidden) return;

    setData(clone(DEFAULT_DATA));

    const canceler = new AbortController();
    const follow = !terminalRunStates.has(trialState);

    consumeStream(
      detApi.StreamingProfiler.getTrialProfilerMetrics(
        trialId,
        labelsName,
        labelsAgentId,
        labelsGpuUuid,
        labelsMetricType,
        follow,
        { signal: canceler.signal },
      ),
      (event: V1GetTrialProfilerMetricsResponse) => {
        setData(prev => {
          if (event.batch.values.length !== 0) {
            for (let i = 0; i < event.batch.values.length; i++) {
              const batch: number = event.batch.batches[i];
              const value: number = event.batch.values[i];
              const time: number = Date.parse(event.batch.timestamps[i] as unknown as string);
              const labelName: string = event.batch.labels.gpuUuid || event.batch.labels.name;

              if (!prev.names.includes(labelName)) prev.names.push(labelName);
              if (!prev.dataByTime[time]) prev.dataByTime[time] = {};
              prev.dataByTime[time].batch = batch;
              prev.dataByTime[time][labelName] = value;
            }

            return {
              ...prev,
              isEmpty: !hasObjectKeys(prev.dataByTime) && !hasObjectKeys(prev.names),
              isLoading: false,
            };
          }
          return prev;
        });
      },
    ).finally(() => {
      setData(prev => ({ ...prev, isLoading: false }));
    });

    return () => canceler.abort();
  }, [
    labelsAgentId,
    labelsGpuUuid,
    labelsMetricType,
    labelsName,
    trialId,
    trialState,
    ui.isPageHidden,
  ]);

  return data;
};
