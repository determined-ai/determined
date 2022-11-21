import { useEffect, useState } from 'react';

import { V1MetricNamesResponse } from 'services/api-ts-sdk/models';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import { alphaNumericSorter } from 'shared/utils/sort';
import { Metric, MetricType } from 'types';

export interface UseMetricsInterface {
  errorHandler: () => void;
  experimentId: number;
}

const useMetricNames = (experimentId: number, errorHandler: (e: unknown) => void): Metric[] => {
  const [metrics, setMetrics] = useState<Metric[]>([]);

  useEffect(() => {
    const canceler = new AbortController();
    const trainingMetricsMap: Record<string, boolean> = {};
    const validationMetricsMap: Record<string, boolean> = {};

    readStream<V1MetricNamesResponse>(
      detApi.Internal.metricNames(experimentId, undefined, {
        signal: canceler.signal,
      }),
      (event: V1MetricNamesResponse) => {
        if (!event) return;
        /*
         * The metrics endpoint can intermittently send empty lists,
         * so we keep track of what we have seen on our end and
         * only add new metrics we have not seen to the list.
         */
        (event.trainingMetrics || []).forEach((metric) => (trainingMetricsMap[metric] = true));
        (event.validationMetrics || []).forEach((metric) => (validationMetricsMap[metric] = true));
        const newTrainingMetrics = Object.keys(trainingMetricsMap).sort(alphaNumericSorter);
        const newValidationMetrics = Object.keys(validationMetricsMap).sort(alphaNumericSorter);
        const newMetrics = [
          ...newValidationMetrics.map((name) => ({ name, type: MetricType.Validation })),
          ...newTrainingMetrics.map((name) => ({ name, type: MetricType.Training })),
        ];
        setMetrics((prevMetrics) =>
          prevMetrics.length === newMetrics.length ? prevMetrics : newMetrics,
        );
      },
      errorHandler,
    );
    return () => canceler.abort();
  }, [experimentId, errorHandler]);
  return metrics;
};

export default useMetricNames;
