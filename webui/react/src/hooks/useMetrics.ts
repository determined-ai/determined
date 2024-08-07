import { makeToast } from 'hew/Toast';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import _ from 'lodash';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { terminalRunStates } from 'constants/states';
import useMetricNames from 'hooks/useMetricNames';
import usePolling from 'hooks/usePolling';
import { timeSeries } from 'services/api';
import {
  FlatRun,
  Metric,
  MetricContainer,
  RunState,
  Scale,
  Serie,
  TrialDetails,
  TrialSummary,
  XAxisDomain,
} from 'types';
import handleError, { ErrorType } from 'utils/error';
import { metricToKey } from 'utils/metric';
import { isRun } from 'utils/run';

type MetricName = string;
export interface RunMetrics {
  data: Record<MetricName, Serie>;
  metrics: Metric[];
}

export interface RunMetricData {
  data: Record<number, Record<string, Serie>>;
  isLoaded: boolean;
  metrics: Metric[];
  scale: Scale;
  setScale: React.Dispatch<React.SetStateAction<Scale>>;
  metricHasData: Record<string, boolean>;
}

const summarizedMetricToSeries = (
  allDownsampledMetrics: MetricContainer[],
  selectedMetrics: Metric[],
): {
  data: Record<MetricName, Serie>;
  metricHasData: Record<MetricName, boolean>;
} => {
  const rawBatchValuesMap: Record<string, [number, number][]> = {};
  const rawBatchTimesMap: Record<string, [number, number][]> = {};
  const rawBatchEpochMap: Record<string, [number, number][]> = {};
  allDownsampledMetrics.forEach((summMetric) => {
    summMetric.data.forEach((avgMetrics) => {
      selectedMetrics.forEach((metric) => {
        if (summMetric.group !== metric.group) return;

        const metricKey = metricToKey(metric);
        const value = avgMetrics.values[metric.name];
        if (!rawBatchValuesMap[metricKey]) rawBatchValuesMap[metricKey] = [];
        if (!rawBatchTimesMap[metricKey]) rawBatchTimesMap[metricKey] = [];
        if (!rawBatchEpochMap[metricKey]) rawBatchEpochMap[metricKey] = [];

        if (value || value === 0) {
          rawBatchValuesMap[metricKey]?.push([avgMetrics.batches, value]);
          if (avgMetrics.time)
            rawBatchTimesMap[metricKey]?.push([new Date(avgMetrics.time).getTime() / 1000, value]);
          if (!_.isUndefined(avgMetrics.epoch))
            rawBatchEpochMap[metricKey]?.push([avgMetrics.epoch, value]);
        }
      });
    });
  });
  const recordData: Record<string, Serie> = {};
  const metricHasData: Record<string, boolean> = {};
  selectedMetrics.forEach((metric) => {
    const metricKey = metricToKey(metric);
    const data: Partial<Record<XAxisDomain, [number, number][]>> = {};
    if (rawBatchValuesMap[metricKey]) data[XAxisDomain.Batches] = rawBatchValuesMap[metricKey];
    if (rawBatchTimesMap[metricKey]) data[XAxisDomain.Time] = rawBatchTimesMap[metricKey];
    if (rawBatchEpochMap[metricKey]) data[XAxisDomain.Epochs] = rawBatchEpochMap[metricKey];

    const series: Serie = {
      data,
      name: `${metric.group}.${metric.name}`,
    };

    recordData[metricToKey(metric)] = series;
  });
  const xAxisOptions = Object.values(XAxisDomain);
  // Record whether or not each metric contains at least one value for any
  // xAxis option.
  Object.keys(recordData).forEach((key) => {
    metricHasData[key] ||= xAxisOptions.some(
      (xAxis) => (recordData?.[key]?.data?.[xAxis]?.length ?? 0) > 0,
    );
  });
  return { data: recordData, metricHasData };
};

// consistent reference to prevent extra calls
const EMPTY_METRICS: Metric[] = [];

export const useMetrics = (records: (TrialDetails | FlatRun | undefined)[]): RunMetricData => {
  const recordsAllTerminated = records?.every((record) =>
    terminalRunStates.has(record?.state ?? RunState.Active),
  );
  const recordsAllNonTerminal = !records?.find((record) =>
    terminalRunStates.has(record?.state ?? RunState.Error),
  );
  const experimentIds = useMemo(
    () =>
      records.flatMap((r) =>
        r === undefined
          ? []
          : isRun(r)
            ? r.experiment === undefined
              ? []
              : [r.experiment.id]
            : [r.experimentId],
      ),
    [records],
  );
  const handleMetricNamesError = useCallback(
    (e: unknown) => {
      handleError(e, {
        publicMessage: `Failed to load metric names for records ${records?.map(
          (r) => `[${r?.id}]`,
        )}.`,
        publicSubject: 'Metric name stream failed.',
        type: ErrorType.Api,
      });
    },
    [records],
  );

  const loadableMetrics = useMetricNames(
    experimentIds,
    handleMetricNamesError,
    recordsAllNonTerminal,
  );
  const metrics = useMemo(() => {
    return loadableMetrics
      .map((m) => (m.length === 0 ? EMPTY_METRICS : m))
      .getOrElse(EMPTY_METRICS);
  }, [loadableMetrics]);
  const [scale, setScale] = useState<Scale>(Scale.Linear);
  const [curResponse, setCurResponse] = useState<Loadable<TrialSummary[]>>(NotLoaded);

  // don't replace this with usePrevious -- we need the ref to prevent
  // fetchRecordSummary from regenerating when the previous value is detected
  const recordIdsRef = useRef<number[]>([]);
  const curRecordIds = useMemo(() => {
    const ids = records
      ?.map((t) => t?.id)
      .filter(<T>(i: T): i is Exclude<T, undefined> => i !== undefined);
    return _.isEqual(ids, recordIdsRef.current) ? recordIdsRef.current : ids;
  }, [records]);

  const fetchRecordSummary = useCallback(async () => {
    if (loadableMetrics.isNotLoaded) return;
    if (curRecordIds.length === 0) {
      // If there are no trials selected then
      // no data is available.
      setCurResponse(Loaded([]));
      return;
    }
    if (curRecordIds.length > 0) {
      const prevRecordIds = recordIdsRef.current;
      // If the record ids have not changed then we do not need to
      // show the loading state again.
      if (curRecordIds !== prevRecordIds) setCurResponse(NotLoaded);
      try {
        const response = await timeSeries({
          maxDatapoints: screen.width > 1600 ? 1500 : 1000,
          metrics,
          startBatches: 0,
          trialIds: curRecordIds,
        });
        setCurResponse((prev) => {
          const loadedData = Loaded(response);
          return _.isEqual(loadedData, prev) ? prev : loadedData;
        });
      } catch (e) {
        makeToast({ severity: 'Error', title: 'Error fetching metrics' });
      }
    }
  }, [loadableMetrics, metrics, curRecordIds]);
  useEffect(() => {
    recordIdsRef.current = curRecordIds;
  });
  const requestData = useMemo(() => {
    return Loadable.all([curResponse, loadableMetrics]);
  }, [curResponse, loadableMetrics]);
  const [metricHasData, data] = useMemo(() => {
    return requestData
      .map(([response, metrics]) => {
        const metricsHaveData: Record<string, boolean> = {};
        const newData: Record<number, Record<string, Serie>> = {};
        response.forEach((r) => {
          const { data: recordData, metricHasData } = summarizedMetricToSeries(r?.metrics, metrics);
          Object.keys(metricHasData).forEach((key) => {
            metricsHaveData[key] ||= metricHasData[key];
          });
          newData[r.id] = recordData;
        });
        return [metricsHaveData, newData] as const;
      })
      .getOrElse([{}, {}]);
  }, [requestData]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([fetchRecordSummary()]);
  }, [fetchRecordSummary]);

  const { stopPolling } = usePolling(fetchAll, { interval: 2000, rerunOnNewFn: true });

  useEffect(() => {
    if (recordsAllTerminated) {
      stopPolling();
    }
  }, [recordsAllTerminated, stopPolling]);

  if (recordsAllTerminated) {
    stopPolling();
  }

  return {
    data,
    isLoaded: requestData.isLoaded,
    metricHasData,
    metrics,
    scale,
    setScale,
  };
};
