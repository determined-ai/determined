import { useEffect, useState } from 'react';

import { terminalRunStates } from 'constants/states';
import { useStore } from 'contexts/Store';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import { RunState } from 'types';
import { clone } from 'utils/data';

import { MetricsAggregateInterface, MetricType, ProfilerMetricsResponse } from './types';

const BUCKET_SIZE = .1; // in seconds
const BUCKET_WIDTH = 1000 * BUCKET_SIZE; // seconds -> millisecondss
const BUFFER_WINDOW = 5; // in seconds
const BUFFER_BUCKETS = BUFFER_WINDOW / BUCKET_SIZE;
const INIT_BUCKETS = BUFFER_BUCKETS + 1;
const BATCH_INDEX = 1;

const DEFAULT_DATA: MetricsAggregateInterface = {
  isEmpty: true,
  isLoading: true,
  names: [ 'batch' ],
};

const parseTime = (time: string): number => Math.floor(Date.parse(time) / 100) * 100;

const getIndexForTimestamp = (initialTimestamp: number, timestamp: number) =>
  Math.floor((timestamp - initialTimestamp) / BUCKET_WIDTH) + BUFFER_BUCKETS;

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
        setData(prev => {
          const newData = event.batch;
          if (newData.values.length !== 0) {
            let names = prev.names;
            const newTimestamps = newData.timestamps;
            const initialTimestamp =
              prev.initialTimestamp ?? parseTime((newTimestamps[0]));
            let data = prev.data;
            const labelName: string = newData.labels.gpuUuid || newData.labels.name;
            if (data == null) {
              data = [
                Array(INIT_BUCKETS)
                  .fill(null)
                  .map((_, i) => (i - BUFFER_BUCKETS) * BUCKET_WIDTH + initialTimestamp),
                Array(INIT_BUCKETS).fill(null),
              ];
            }
            let labelRowIndex = names.indexOf(labelName) + 1; // 0th is time
            if (labelRowIndex === 0) {
              labelRowIndex = names.length;
              names = [ ...names, labelName ];
              data = [ ...data, data[0].map(() => null) ];
            }

            // const newDataMaxTimestamp = Math.max(...newTimestamps.map(parseTime));
            // console.log(newTimestamps.map((ts, i) => !(ts > data[0][i + 1])).every(x => x));
            const timestamps = data[0];
            const prevMaxTimestamp = timestamps[timestamps.length - 1] ?? Number.MAX_SAFE_INTEGER;
            const newMaxTimestamp = parseTime(
              (newTimestamps[newTimestamps.length - 1]),
            );

            if (prevMaxTimestamp < newMaxTimestamp) {
              for (
                let ts = prevMaxTimestamp + BUCKET_WIDTH;
                ts <= newMaxTimestamp;
                ts += BUCKET_WIDTH
              ) {
                timestamps.push(ts);
              }

              for (let i = 1; i < data.length; i++) {
                const series = data[i] as (number | null | undefined)[]; // the type of ySeries
                while (series.length < timestamps.length)
                  series.push(null);
              }
            }

            for (let i = 0; i < newData.values.length; i++) {
              const batch: number = newData.batches[i];
              const value: number = newData.values[i];
              const timestamp = parseTime(newData.timestamps[i]);

              const timestampIndex = getIndexForTimestamp(initialTimestamp, timestamp);

              data[BATCH_INDEX][timestampIndex] = batch;
              data[labelRowIndex][timestampIndex] = value;

            }

            return {
              ...prev,
              data: [ ...data ],
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
