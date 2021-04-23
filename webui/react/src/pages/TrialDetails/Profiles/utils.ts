import { useEffect, useState } from 'react';
import { debounce } from 'throttle-debounce';
import { AlignedData } from 'uplot';

import {
  V1GetTrialProfilerAvailableSeriesResponse, V1GetTrialProfilerMetricsResponse,
} from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';

export enum MetricType {
  System = 'PROFILER_METRIC_TYPE_SYSTEM',
  Timing = 'PROFILER_METRIC_TYPE_TIMING',
}

// {[metric_type]: {[name]: {[agent]: [gpu, ..], ..}, ..}, ..}
export type AvailableSeries = Record<string, Record<string, Record<string, string[]>>>;

export type MetricsAggregateInterface = {
  // group information by {[batch]: {[name]: value, ..}, ..}
  dataByBatch: Record<number, Record<string, number>>,
  // group information by {[time]: {[name]: value, ..}, ..}
  dataByUnixTime: Record<number, Record<string, number>>,
  // set to false when the 1st event is received
  isLoading: boolean,
  // names to ease building the chart later
  names: string[],
};

export const convertMetricsToUplotData =
  (data: Record<number, Record<string, number>>): AlignedData => {
    const series: Array<number[]> = [];
    const timeSerie: number[] = [];

    Object.entries(data).forEach(([ timeString, nameList ]) => {
      timeSerie.push(parseInt(timeString));

      Object.keys(nameList).forEach((name, nameIndex) => {
        if (!series[nameIndex]) {
          series[nameIndex] = [];
        }

        series[nameIndex].push(nameList[name]);
      });
    });

    return [ timeSerie, ...series ];
  };

export const getUnitForMetricName = (metricName: string): string => {
  if (metricName === 'cpu_util_simple') return '%';
  if (metricName === 'disk_iops') return '';
  if (metricName === 'disk_throughput_read') return 'bytes/second';
  if (metricName === 'disk_throughput_write') return 'bytes/second';
  if (metricName === 'free_memory') return 'Gigabytes';
  if (metricName === 'gpu_util') return '%';
  if (metricName === 'net_throughput_recv') return 'Gigabit/s';
  if (metricName === 'net_throughput_sent') return 'Gigabit/s';
  return '';
};

export const useFetchAvailableSeries = (trialId: number): AvailableSeries => {
  const [ availableSeries, setAvailableSeries ] = useState<AvailableSeries>({});

  useEffect(() => {
    const canceler = new AbortController();

    consumeStream(
      detApi.StreamingUnimplemented.determinedGetTrialProfilerAvailableSeries(
        trialId,
        { signal: canceler.signal },
      ),
      (event: V1GetTrialProfilerAvailableSeriesResponse) => {
        const newAvailableSeries: AvailableSeries = {};

        event.labels.forEach(label => {
          const agentId: string = label.agentId as unknown as string;
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

export const useFetchMetrics = (
  trialId: number,
  labelsMetricType: MetricType,
  labelsName: string|undefined = undefined,
  labelsAgentId: string|undefined = undefined,
  labelsGpuUuid: string|undefined = undefined,
): MetricsAggregateInterface => {
  const [ data, setData ] = useState<MetricsAggregateInterface>({
    dataByBatch: {},
    dataByUnixTime: {},
    isLoading: true,
    names: [],
  });

  useEffect(() => {
    const broadcastUpdate = debounce(2000, (fnData: MetricsAggregateInterface) => {
      setData({
        dataByBatch: { ...fnData.dataByBatch },
        dataByUnixTime: { ...fnData.dataByUnixTime },
        isLoading: false,
        names: fnData.names,
      });
    });
    const canceler = new AbortController();
    const internalData: MetricsAggregateInterface = {
      dataByBatch: {},
      dataByUnixTime: {},
      isLoading: true,
      names: [],
    };

    // reset data
    setData(internalData);

    consumeStream(
      detApi.StreamingUnimplemented.determinedGetTrialProfilerMetrics(
        trialId,
        labelsName,
        labelsAgentId,
        labelsGpuUuid,
        labelsMetricType,
        { signal: canceler.signal },
      ),
      (event: V1GetTrialProfilerMetricsResponse) => {
        const labelName: string = event.batch.labels.name;

        if (!internalData.names.includes(labelName)) {
          internalData.names = [ ...internalData.names, labelName ];
        }

        event.batch.values.forEach((v, index) => {
          const value: number = event.batch.values[index];

          const batch: number = event.batch.batches[index];
          if (!internalData.dataByBatch[batch]) {
            internalData.dataByBatch[batch] = {};
          }
          internalData.dataByBatch[batch][labelName] = value;

          const unixTime: number =
            Date.parse(event.batch.timestamps[index] as unknown as string) / 1000;
          if (!internalData.dataByUnixTime[unixTime]) {
            internalData.dataByUnixTime[unixTime] = {};
          }
          internalData.dataByUnixTime[unixTime][labelName] = value;
        });

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
