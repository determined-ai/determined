import { useEffect, useState } from 'react';

import type { Serie } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import useUI from 'components/kit/Theme';
import { terminalRunStates } from 'constants/states';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import { RunState } from 'types';

import { MetricsAggregateInterface, MetricType, ProfilerMetricsResponse } from './types';

const DEFAULT_DATA: MetricsAggregateInterface = {
  data: [],
  isEmpty: true,
  isLoading: true,
  names: [],
};

/* Get the time as the nearest 1/10th of a second timestamp */
const parseTime = (time: string): number => Math.floor(Date.parse(time) / 100) / 10;

export const useFetchProfilerMetrics = (
  trialId: number,
  trialState: RunState,
  labelsMetricType: MetricType,
  labelsName: string | undefined = undefined,
  labelsAgentId: string | undefined = undefined,
  labelsGpuUuid: string | undefined = undefined,
): MetricsAggregateInterface => {
  const { ui } = useUI();
  const [data, setData] = useState<MetricsAggregateInterface>(structuredClone(DEFAULT_DATA));

  useEffect(() => {
    if (ui.isPageHidden) return;

    setData(structuredClone(DEFAULT_DATA));

    const canceler = new AbortController();
    const follow = !terminalRunStates.has(trialState);

    readStream(
      detApi.StreamingProfiler.getTrialProfilerMetrics(
        trialId,
        labelsName,
        labelsAgentId,
        labelsGpuUuid,
        labelsMetricType,
        follow,
        { signal: canceler.signal },
      ),
      (event: ProfilerMetricsResponse) => {
        setData((prev) => {
          const newData = event.batch;
          if (newData.values.length !== 0) {
            const names = prev.names;
            const newTimestamps = newData.timestamps;
            const initialTimestamp = prev.initialTimestamp ?? parseTime(newTimestamps[0]);
            const seriesMap: Map<string, Serie> = new Map();
            const serieData = prev.data;
            const labelName: string = newData.labels.gpuUuid || newData.labels.name;

            if (serieData.length >= 0) {
              for (let i = 0; i < serieData.length; i++) {
                seriesMap.set(names[i], serieData[i]);
              }
            }

            if (!seriesMap.has(labelName)) {
              const s_new: Serie = { data: { [XAxisDomain.Time]: [] }, name: labelName };
              seriesMap.set(labelName, s_new);
              names.push(labelName);
            }

            for (let i = 0; i < newData.values.length; i++) {
              const value: number = newData.values[i];
              const timestamp = parseTime(newData.timestamps[i]);

              const timeSerie = seriesMap.get(labelName);
              if (timeSerie) {
                timeSerie.data[XAxisDomain.Time]?.push([timestamp, value]);
                seriesMap.set(labelName, timeSerie);
              }
            }

            return {
              ...prev,
              data: Array.from(seriesMap.values()),
              initialTimestamp,
              isEmpty: false,
              isLoading: false,
              names,
            };
          }
          return prev;
        });
      },
    ).finally(() => {
      setData((prev) => ({ ...prev, isLoading: false }));
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
