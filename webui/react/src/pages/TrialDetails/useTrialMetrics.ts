import _ from 'lodash';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { Serie, TRAINING_SERIES_COLOR, VALIDATION_SERIES_COLOR } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import { terminalRunStates } from 'constants/states';
import useMetricNames from 'hooks/useMetricNames';
import usePolling from 'hooks/usePolling';
import usePrevious from 'hooks/usePrevious';
import { timeSeries } from 'services/api';
import { Metric, MetricContainer, MetricType, RunState, Scale, TrialDetails } from 'types';
import { message } from 'utils/dialogApi';
import handleError, { ErrorType } from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { metricToKey } from 'utils/metric';

type MetricName = string;
export interface TrialMetrics {
  data: Record<MetricName, Serie>;
  metrics: Metric[];
}

export interface TrialMetricData {
  data: Record<number, Record<string, Serie>>;
  isLoaded: boolean;
  metrics: Metric[];
  scale: Scale;
  setScale: React.Dispatch<React.SetStateAction<Scale>>;
  metricHasData: Record<string, boolean>;
  selectedMetrics: Metric[];
}

const summarizedMetricToSeries = (
  allDownsampledMetrics: MetricContainer[],
  selectedMetrics: Metric[],
): {
  data: Record<MetricName, Serie>;
  metricHasData: Record<MetricName, boolean>;
  selectedMetrics: Metric[];
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
  const trialData: Record<string, Serie> = {};
  const metricHasData: Record<string, boolean> = {};
  selectedMetrics.forEach((metric) => {
    const metricKey = metricToKey(metric);
    const data: Partial<Record<XAxisDomain, [number, number][]>> = {};
    if (rawBatchValuesMap[metricKey]) data[XAxisDomain.Batches] = rawBatchValuesMap[metricKey];
    if (rawBatchTimesMap[metricKey]) data[XAxisDomain.Time] = rawBatchTimesMap[metricKey];
    if (rawBatchEpochMap[metricKey]) data[XAxisDomain.Epochs] = rawBatchEpochMap[metricKey];

    const series: Serie = {
      color:
        metric.group === MetricType.Validation ? VALIDATION_SERIES_COLOR : TRAINING_SERIES_COLOR,
      data,
      metricType: metric.group,
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
  return { data: trialData, metricHasData, selectedMetrics };
};

export const useTrialMetrics = (trials: (TrialDetails | undefined)[]): TrialMetricData => {
  const trialsAllTerminated = trials?.every((trial) =>
    terminalRunStates.has(trial?.state ?? RunState.Active),
  );
  const trialsAllNonTerminal = !trials?.find((trial) =>
    terminalRunStates.has(trial?.state ?? RunState.Error),
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

  const loadableMetrics = useMetricNames(
    experimentIds,
    handleMetricNamesError,
    trialsAllNonTerminal,
  );
  const metricNamesLoaded = Loadable.isLoaded(loadableMetrics);
  const metrics = useMemo(() => {
    return Loadable.getOrElse([], loadableMetrics);
  }, [loadableMetrics]);
  const [loadableData, setLoadableData] =
    useState<Loadable<Record<number, Record<string, Serie>>>>(NotLoaded);
  const [metricHasData, setMetricHasData] = useState<Record<string, boolean>>({});
  const [scale, setScale] = useState<Scale>(Scale.Linear);
  const [selectedMetrics, setSelectedMetrics] = useState<Metric[]>([]);

  const previousTrials = usePrevious(trials, []);

  const fetchTrialSummary = useCallback(async () => {
    // If the trial ids have not changed then we do not need to
    // show the loading state again.
    if (!_.isEqual(previousTrials, trials)) setLoadableData(NotLoaded);

    if (trials.length === 0) {
      // If there are no trials selected then
      // no data is available.
      setMetricHasData({});
      setLoadableData(Loaded({}));
      return;
    }
    if (trials.length > 0) {
      try {
        const metricsHaveData: Record<string, boolean> = {};
        const response = await timeSeries({
          maxDatapoints: screen.width > 1600 ? 1500 : 1000,
          metrics,
          startBatches: 0,
          trialIds: trials?.map((t) => t?.id || 0).filter((i) => i > 0),
        });
        const newData: Record<number, Record<string, Serie>> = {};
        response.forEach((r) => {
          const {
            data: trialData,
            metricHasData,
            selectedMetrics: s,
          } = summarizedMetricToSeries(r?.metrics, metrics);
          Object.keys(metricHasData).forEach((key) => {
            metricsHaveData[key] ||= metricHasData[key];
          });
          newData[r.id] = trialData;
          setSelectedMetrics((prev) => (_.isEqual(selectedMetrics, s) ? prev : s));
        });
        setLoadableData((prev) =>
          _.isEqual(Loadable.getOrElse([], prev), newData) ? prev : Loaded(newData),
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
  }, [loadableMetrics, metrics, selectedMetrics, trials, previousTrials]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([fetchTrialSummary()]);
  }, [fetchTrialSummary]);

  const { stopPolling } = usePolling(fetchAll, { interval: 2000, rerunOnNewFn: true });

  useEffect(() => {
    if (trialsAllTerminated) {
      stopPolling();
    }
  }, [trialsAllTerminated, stopPolling]);

  if (trialsAllTerminated) {
    stopPolling();
  }

  return {
    data: Loadable.getOrElse({}, loadableData),
    isLoaded: metricNamesLoaded && Loadable.isLoaded(loadableData),
    metricHasData,
    metrics,
    scale,
    selectedMetrics,
    setScale,
  };
};
