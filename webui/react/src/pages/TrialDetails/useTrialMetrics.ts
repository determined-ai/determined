import { useCallback, useEffect, useState } from 'react';

import { Serie, TRAINING_SERIES_COLOR, VALIDATION_SERIES_COLOR } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import { terminalRunStates } from 'constants/states';
import useMetricNames from 'hooks/useMetricNames';
import { timeSeries } from 'services/api';
import usePolling from 'shared/hooks/usePolling';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { Metric, MetricContainer, MetricType, RunState, Scale, TrialDetails } from 'types';
import handleError from 'utils/error';
import { metricToKey } from 'utils/metric';

type MetricName = string;
export interface TrialMetrics {
  data: Record<MetricName, Serie>;
  metrics: Metric[];
}
const summarizedMetricToSeries = (
  allDownsampledMetrics: MetricContainer[],
  selectedMetrics: Metric[],
): Record<string, Serie> => {
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

        if (value) {
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
  selectedMetrics.forEach((metric) => {
    const data: Partial<Record<XAxisDomain, [number, number][]>> = {
      [XAxisDomain.Batches]: rawBatchValuesMap[metric.name],
      [XAxisDomain.Time]: rawBatchTimesMap[metric.name],
      [XAxisDomain.Epochs]: rawBatchEpochMap[metric.name],
    };
    const series: Serie = {
      color:
        metric.type === MetricType.Validation ? VALIDATION_SERIES_COLOR : TRAINING_SERIES_COLOR,
      data,
      metricType: metric.type,
      name: metric.name,
    };
    trialData[metricToKey(metric)] = series;
  });

  return trialData;
};
export const useTrialMetrics = (
  trial: TrialDetails | undefined,
): {
  data: Record<string, Serie> | undefined;
  metrics: Metric[];
  scale: Scale;
  setScale: React.Dispatch<React.SetStateAction<Scale>>;
} => {
  const trialTerminated = terminalRunStates.has(trial?.state ?? RunState.Active);

  const handleMetricNamesError = useCallback(
    (e: unknown) => {
      handleError(e, {
        publicMessage: `Failed to load metric names for trial ${trial?.id}.`,
        publicSubject: 'Experiment metric name stream failed.',
        type: ErrorType.Api,
      });
    },
    [trial?.id],
  );

  const metrics = useMetricNames(trial?.experimentId, handleMetricNamesError);
  const [data, setData] = useState<Record<MetricName, Serie>>();
  const [scale, setScale] = useState<Scale>(Scale.Linear);

  const fetchTrialSummary = useCallback(async () => {
    if (trial?.id) {
      const response = await timeSeries({
        maxDatapoints: screen.width > 1600 ? 1500 : 1000,
        metricNames: metrics,
        scale: scale,
        startBatches: 0,
        trialIds: [trial?.id],
      });

      setData((prev) => {
        if (isEqual(prev, response)) return prev;
        const trialData = summarizedMetricToSeries(response[0]?.metrics, metrics);
        return trialData;
      });
    }
  }, [metrics, trial?.id, scale]);

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

  return { data, metrics, scale, setScale };
};
