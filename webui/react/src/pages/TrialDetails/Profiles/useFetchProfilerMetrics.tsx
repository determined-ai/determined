import { useEffect, useState } from 'react';

import { terminalRunStates } from 'constants/states';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import useUI from 'shared/contexts/stores/UI';
import { clone } from 'shared/utils/data';
import { RunState } from 'types';

import { MetricsAggregateInterface, MetricType, ProfilerMetricsResponse } from './types';

const BUCKET_SIZE = 0.1; // in seconds
const BUCKET_WIDTH = 1000 * BUCKET_SIZE; // seconds -> millisecondss
const PADDING_WINDOW = 5; // in seconds
const PADDING_BUCKETS = PADDING_WINDOW / BUCKET_SIZE;
const INIT_BUCKETS = PADDING_BUCKETS + 1; // plus the one for the initial timestamp
const BATCH_INDEX = 1;

const DEFAULT_DATA: MetricsAggregateInterface = {
  isEmpty: true,
  isLoading: true,
  names: ['batch'],
};

/* Get the time as the nearest 1/10th of a second timestamp */
const parseTime = (time: string): number => Math.floor(Date.parse(time) / 100) * 100;

const getIndexForTimestamp = (initialTimestamp: number, timestamp: number) =>
  Math.floor((timestamp - initialTimestamp) / BUCKET_WIDTH) + PADDING_BUCKETS;

export const useFetchProfilerMetrics = (
  trialId: number,
  trialState: RunState,
  labelsMetricType: MetricType,
  labelsName: string | undefined = undefined,
  labelsAgentId: string | undefined = undefined,
  labelsGpuUuid: string | undefined = undefined,
): MetricsAggregateInterface => {
  const { ui } = useUI();
  const [data, setData] = useState<MetricsAggregateInterface>(clone(DEFAULT_DATA));

  useEffect(() => {
    if (ui.isPageHidden) return;

    setData(clone(DEFAULT_DATA));

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
            let names = prev.names;
            const newTimestamps = newData.timestamps;
            const initialTimestamp = prev.initialTimestamp ?? parseTime(newTimestamps[0]);
            let data = prev.data;
            const labelName: string = newData.labels.gpuUuid || newData.labels.name;
            if (data == null) {
              data = [
                Array(INIT_BUCKETS)
                  .fill(null)
                  .map((_, i) => (i - PADDING_BUCKETS) * BUCKET_WIDTH + initialTimestamp),
                Array(INIT_BUCKETS).fill(null),
              ];
            }

            /**
             * data is [xSeries, ...ySeries[]], and names
             * corresponds to ySeries[], so we offset by 1
             * to get the index in data
             */
            const labelNamesIndex = names.indexOf(labelName);
            let labelDataIndex;

            if (labelNamesIndex === -1) {
              names = [...names, labelName];
              data = [...data, data[0].map(() => null)];
              labelDataIndex = data.length - 1;
            } else {
              labelDataIndex = labelNamesIndex + 1;
            }

            const timestamps = data[0];
            const prevMaxTimestamp = timestamps[timestamps.length - 1] ?? Number.MAX_SAFE_INTEGER;
            const newMaxTimestamp = parseTime(newTimestamps[newTimestamps.length - 1]);

            if (prevMaxTimestamp < newMaxTimestamp) {
              for (
                let ts = prevMaxTimestamp + BUCKET_WIDTH;
                ts <= newMaxTimestamp;
                ts += BUCKET_WIDTH
              ) {
                timestamps.push(ts);
              }

              for (let i = 1; i < data.length; i++) {
                const series = data[i] as (number | null | undefined)[]; // AlignedData.ySeries
                while (series.length < timestamps.length) series.push(null);
              }
            }

            for (let i = 0; i < newData.values.length; i++) {
              const batch: number = newData.batches[i];
              const value: number = newData.values[i];
              const timestamp = parseTime(newData.timestamps[i]);

              const timestampIndex = getIndexForTimestamp(initialTimestamp, timestamp);

              if (timestampIndex >= 0) {
                data[BATCH_INDEX][timestampIndex] = batch;
                data[labelDataIndex][timestampIndex] = value;
              }
            }

            return {
              ...prev,
              data: [...data],
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
