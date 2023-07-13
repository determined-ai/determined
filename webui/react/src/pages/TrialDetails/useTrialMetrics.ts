import { useCallback, useEffect, useMemo, useState } from 'react';

import { Serie, TRAINING_SERIES_COLOR, VALIDATION_SERIES_COLOR } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import { terminalRunStates } from 'constants/states';
import useMetricNames from 'hooks/useMetricNames';
import usePolling from 'hooks/usePolling';
import usePrevious from 'hooks/usePrevious';
import { timeSeries } from 'services/api';
import { Metric, MetricContainer, MetricType, RunState, Scale, TrialDetails } from 'types';
import { isEqual } from 'utils/data';
import { message } from 'utils/dialogApi';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { metricToKey } from 'utils/metric';

type MetricName = string;
export interface TrialMetrics {
  data: Record<MetricName, Serie>;
  metrics: Metric[];
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
        const value = avgMetrics.values[metric.name];
        if (!rawBatchValuesMap[metric.name]) rawBatchValuesMap[metric.name] = [];

        if (!rawBatchTimesMap[metric.name]) rawBatchTimesMap[metric.name] = [];

        if (!rawBatchEpochMap[metric.name]) rawBatchEpochMap[metric.name] = [];

        if (value || value === 0) {
          rawBatchValuesMap[metric.name]?.push([avgMetrics.batches, value]);
          if (avgMetrics.time)
            rawBatchTimesMap[metric.name]?.push([
              new Date(avgMetrics.time).getTime() / 1000,
              value,
            ]);
          if (avgMetrics.epoch) rawBatchEpochMap[metric.name]?.push([avgMetrics.epoch, value]);
        }
      });
    });
  });
  const trialData: Record<string, Serie> = {};
  const metricHasData: Record<string, boolean> = {};
  selectedMetrics.forEach((metric) => {
    const data: Partial<Record<XAxisDomain, [number, number][]>> = {};
    if (rawBatchValuesMap[metric.name]) data[XAxisDomain.Batches] = rawBatchValuesMap[metric.name];
    if (rawBatchTimesMap[metric.name]) data[XAxisDomain.Time] = rawBatchTimesMap[metric.name];
    if (rawBatchEpochMap[metric.name]) data[XAxisDomain.Epochs] = rawBatchEpochMap[metric.name];

    const series: Serie = {
      color:
        metric.type === MetricType.Validation ? VALIDATION_SERIES_COLOR : TRAINING_SERIES_COLOR,
      data,
      metricType: metric.type,
      name: metric.name,
    };
    trialData[metricToKey(metric)] = series;
  });
  const xAxisOptions = Object.values(XAxisDomain);
  // Record whether or not each metric contains at least one value for any
  // xAxis option.
  Object.keys(trialData).forEach((key) => {
    metricHasData[key] ||= xAxisOptions.some(
      (xAxis) => (trialData?.[key]?.data?.[xAxis]?.length ?? 0) > 0,
    );
  });
  return { data: trialData, metricHasData };
};

export const useTrialMetrics = (
  trials: (TrialDetails | undefined)[],
): {
  data: Record<number, Record<string, Serie>>;
  isLoaded: boolean;
  metrics: Metric[];
  scale: Scale;
  setScale: React.Dispatch<React.SetStateAction<Scale>>;
  metricHasData: Record<string, boolean>;
} => {
  const trialTerminated = trials?.every((trial) =>
    terminalRunStates.has(trial?.state ?? RunState.Active),
  );
  const experimentIds = useMemo(
    () => trials?.map((t) => t?.experimentId || 0).filter((i) => i > 0),
    [trials],
  );
  const handleMetricNamesError = useCallback(
    (e: unknown) => {
      handleError(e, {
        publicMessage: `Failed to load metric names for trials ${trials?.map(
          (t) => `[${t?.id}]`,
        )}.`,
        publicSubject: 'Experiment metric name stream failed.',
        type: ErrorType.Api,
      });
    },
    [trials],
  );

  const loadableMetrics = useMetricNames(experimentIds, handleMetricNamesError);
  const metricNamesLoaded = Loadable.isLoaded(loadableMetrics);
  const metrics = Loadable.getOrElse([], loadableMetrics);
  const [loadableData, setLoadableData] =
    useState<Loadable<Record<number, Record<string, Serie>>>>(NotLoaded);
  const [scale, setScale] = useState<Scale>(Scale.Linear);
  const [metricHasData, setMetricHasData] = useState<Record<string, boolean>>({});

  const previousTrials = usePrevious(trials, []);

  const fetchTrialSummary = useCallback(async () => {
    // If the trial ids have not changed then we do not need to
    // show the loading state again.
    if (!isEqual(previousTrials, trials)) setLoadableData(NotLoaded);

    if (trials.length === 0) {
      // If there are no trials selected then
      // no data is available.
      return;
    }
    if (trials.length > 0) {
      try {
        const metricsHaveData: Record<string, boolean> = {};
        const response = await timeSeries({
          maxDatapoints: screen.width > 1600 ? 1500 : 1000,
          metricNames: metrics,
          startBatches: 0,
          trialIds: trials?.map((t) => t?.id || 0).filter((i) => i > 0),
        });
        const newData: Record<number, Record<string, Serie>> = {};
        response.forEach((r) => {
          const { data: trialData, metricHasData } = summarizedMetricToSeries(r?.metrics, metrics);
          Object.keys(metricHasData).forEach((key) => {
            metricsHaveData[key] ||= metricHasData[key];
          });
          newData[r.id] = trialData;
        });
        setLoadableData((prev) =>
          isEqual(Loadable.getOrElse([], prev), newData) ? prev : Loaded(newData),
        );
        // Wait until the metric names are loaded
        // to determine if trials have data for any metric
        if (Loadable.isLoaded(loadableMetrics)) {
          setMetricHasData(metricsHaveData);
        }
      } catch (e) {
        message.error('Error fetching metrics');
      }
    }
  }, [metrics, trials, loadableMetrics, previousTrials]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([fetchTrialSummary()]);
  }, [fetchTrialSummary]);

  const { stopPolling } = usePolling(fetchAll, { interval: 2000, rerunOnNewFn: true });

  useEffect(() => {
    if (trialTerminated) {
      stopPolling();
    }
  }, [trialTerminated, stopPolling]);

  if (trialTerminated) {
    stopPolling();
  }

  return {
    data: Loadable.getOrElse({}, loadableData),
    isLoaded: metricNamesLoaded && Loadable.isLoaded(loadableData),
    metricHasData,
    metrics,
    scale,
    setScale,
  };
};
