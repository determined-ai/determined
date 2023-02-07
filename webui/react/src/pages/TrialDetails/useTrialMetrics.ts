import { useCallback, useEffect, useState } from 'react';

import { Serie } from 'components/kit/LineChart';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import { terminalRunStates } from 'constants/states';
import useMetricNames from 'hooks/useMetricNames';
import { compareTrials } from 'services/api';
import usePolling from 'shared/hooks/usePolling';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { Metric, MetricContainer, RunState, TrialDetails } from 'types';
import handleError from 'utils/error';
import { metricToKey } from 'utils/metric';

type MetricName = string;

export interface TrialMetrics {
  data: Record<MetricName, Serie>;
  metrics: Metric[];
}

const summarizedMetricToSeries = (summ: MetricContainer): Serie => {
  const rawData: [number, number][] = [];
  const rawTime: [number, number][] = [];
  const rawEpochs: [number, number][] = [];

  summ.data.forEach((dataPoint) => {
    rawData.push([dataPoint.batches, dataPoint.value]);
  });

  summ.time?.forEach((dataPoint) => {
    rawTime.push([new Date(dataPoint.time).getTime() / 1000, dataPoint.value]);
  });

  const data: Partial<Record<XAxisDomain, [number, number][]>> = {
    [XAxisDomain.Batches]: rawData,
    [XAxisDomain.Time]: rawTime,
  };

  if (summ.epochs) {
    summ.epochs.forEach((dataPoint) => rawEpochs.push([dataPoint.epoch, dataPoint.value]));
    data[XAxisDomain.Epochs] = rawEpochs;
  }

  return {
    data,
    metricType: summ.type,
    name: summ.name,
  };
};

export const useTrialMetrics = (
  trial: TrialDetails | undefined,
): { data: Record<string, Serie> | undefined; metrics: Metric[] } => {
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

  const fetchTrialSummary = useCallback(async () => {
    if (trial?.id) {
      const response = await compareTrials({
        maxDatapoints: screen.width > 1600 ? 1500 : 1000,
        metricNames: metrics,
        // scale: scale,
        startBatches: 0,
        trialIds: [trial?.id],
      });

      setData((prev) => {
        if (isEqual(prev, response)) return prev;
        const trialData = response[0]?.metrics
          .map((summMetric) => {
            const key = metricToKey({ name: summMetric.name, type: summMetric.type });
            return { [key]: summarizedMetricToSeries(summMetric) };
          })
          .reduce((a, b) => ({ ...a, ...b }), {});
        return trialData;
      });
    }
  }, [metrics, trial?.id]);

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

  return { data, metrics };
};
