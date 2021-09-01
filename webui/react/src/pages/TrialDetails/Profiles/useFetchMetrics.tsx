import { useEffect, useState } from 'react';
import { debounce } from 'throttle-debounce';

import { V1GetTrialProfilerMetricsResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';
import { hasObjectKeys } from 'utils/data';

import { MetricsAggregateInterface, MetricType } from './types';

export const useFetchMetrics = (
  trialId: number,
  labelsMetricType: MetricType,
  labelsName: string|undefined = undefined,
  labelsAgentId: string|undefined = undefined,
  labelsGpuUuid: string|undefined = undefined,
): MetricsAggregateInterface => {
  const [ data, setData ] = useState<MetricsAggregateInterface>({
    dataByTime: {},
    isEmpty: true,
    isLoading: true,
    names: [],
  });

  useEffect(() => {
    const broadcastUpdate = debounce(2000, (fnData: MetricsAggregateInterface) => {
      setData({
        dataByTime: { ...fnData.dataByTime },
        isEmpty: !hasObjectKeys(fnData.dataByTime) && !hasObjectKeys(fnData.names),
        isLoading: false,
        names: fnData.names,
      });
    });
    const canceler = new AbortController();
    const internalData: MetricsAggregateInterface = {
      dataByTime: {},
      isEmpty: true,
      isLoading: true,
      names: [ 'batch' ],
    };

    // reset data
    setData(internalData);

    consumeStream(
      detApi.StreamingProfiler.determinedGetTrialProfilerMetrics(
        trialId,
        labelsName,
        labelsAgentId,
        labelsGpuUuid,
        labelsMetricType,
        true,
        { signal: canceler.signal },
      ),
      (event: V1GetTrialProfilerMetricsResponse) => {
        for (let i = 0; i < event.batch.values.length; i++) {
          const batch: number = event.batch.batches[i];
          const value: number = event.batch.values[i];
          const time: number = Date.parse(event.batch.timestamps[i] as unknown as string);
          const labelName: string = event.batch.labels.gpuUuid || event.batch.labels.name;

          if (!internalData.names.includes(labelName)) {
            internalData.names.push(labelName);
          }
          if (!internalData.dataByTime[time]) {
            internalData.dataByTime[time] = {};
          }
          internalData.dataByTime[time].batch = batch;
          internalData.dataByTime[time][labelName] = value;
        }

        broadcastUpdate(internalData);
      },
    );

    return () => {
      broadcastUpdate.cancel();
      canceler.abort();
    };
  }, [ labelsAgentId, labelsGpuUuid, labelsMetricType, labelsName, trialId ]);

  return data;
};
