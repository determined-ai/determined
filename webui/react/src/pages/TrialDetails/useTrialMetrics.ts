import { useCallback, useEffect, useState } from 'react';

import { Serie, TRAINING_SERIES_COLOR, VALIDATION_SERIES_COLOR } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import { terminalRunStates } from 'constants/states';
import useMetricNames from 'hooks/useMetricNames';
import { timeSeries } from 'services/api';
import usePolling from 'shared/hooks/usePolling';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { Metric, MetricType, RunState, Scale, TrialDetails } from 'types';
import handleError from 'utils/error';

type MetricName = string;

export interface TrialMetrics {
  data: Record<MetricName, Serie>;
  metrics: Metric[];
}

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
        const rawBatchValuesMap: Map<string, [number, number][]> = new Map();
        const rawBatchTimesMap: Map<string, [number, number][]> = new Map();
        const rawBatchEpochMap: Map<string, [number, number][]> = new Map();
        response[0]?.metrics.forEach((summMetric) => {
          summMetric.data.forEach((avgMetrics) => {
            metrics.forEach((metric) => {
              const value = avgMetrics.values.get(metric.name);
              if (!rawBatchValuesMap.has(metric.name)) {
                rawBatchValuesMap.set(metric.name, []);
              }
              if (!rawBatchTimesMap.has(metric.name)) {
                rawBatchTimesMap.set(metric.name, []);
              }
              if (!rawBatchEpochMap.has(metric.name)) {
                rawBatchEpochMap.set(metric.name, []);
              }
              if (value) {
                rawBatchValuesMap.get(metric.name)?.push([avgMetrics.batches, value]);
                if (avgMetrics.time)
                  rawBatchTimesMap
                    .get(metric.name)
                    ?.push([new Date(avgMetrics.time).getTime() / 1000, value]);
                if (avgMetrics.epoch)
                  rawBatchEpochMap.get(metric.name)?.push([avgMetrics.epoch, value]);
              }
            });
          });
        });
        const trialData: Record<string, Serie> = {};
        metrics.forEach((metric) => {
          const data: Partial<Record<XAxisDomain, [number, number][]>> = {
            [XAxisDomain.Batches]: rawBatchValuesMap.get(metric.name),
            [XAxisDomain.Time]: rawBatchTimesMap.get(metric.name),
            [XAxisDomain.Epochs]: rawBatchEpochMap.get(metric.name),
          };
          const series: Serie = {
            color:
              metric.type === MetricType.Validation
                ? VALIDATION_SERIES_COLOR
                : TRAINING_SERIES_COLOR,
            data,
            metricType: metric.type,
            name: metric.name,
          };
          trialData[metric.name] = series;
        });
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
