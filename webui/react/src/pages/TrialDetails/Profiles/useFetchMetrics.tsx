import { useEffect, useState } from 'react';

import { terminalRunStates } from 'constants/states';
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
  const [ canceler ] = useState(new AbortController());
  const [ data, setData ] = useState<MetricsAggregateInterface>(clone(DEFAULT_DATA));

  useEffect(() => {
    setData(clone(DEFAULT_DATA));

    consumeStream(
      detApi.StreamingProfiler.getTrialProfilerMetrics(
        trialId,
        labelsName,
        labelsAgentId,
        labelsGpuUuid,
        labelsMetricType,
        true,
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
  }, [ canceler, labelsAgentId, labelsGpuUuid, labelsMetricType, labelsName, trialId ]);

  // Cancel fetch request if trial has reached terminal state.
  useEffect(() => {
    if (terminalRunStates.has(trialState)) {
      setTimeout(() => {
        if (!canceler.signal.aborted) canceler.abort();
      }, 2000);
    }
  }, [ canceler, trialState ]);

  return data;
};
