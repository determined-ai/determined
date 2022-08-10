import { useEffect } from 'react';

import { V1MetricNamesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import { alphaNumericSorter } from 'shared/utils/sort';
import { MetricName, MetricType } from 'types';

export interface UseMetricNamesInterface {
  errorHandler: () => void;
  experimentId: number;
  metricNames: MetricName[];
  setMetricNames: (metrics: MetricName[]) => void;
}

const useMetricNames = (args: UseMetricNamesInterface): void => {
  useEffect(() => {
    const canceler = new AbortController();
    const trainingMetricsMap: Record<string, boolean> = {};
    const validationMetricsMap: Record<string, boolean> = {};

    readStream<V1MetricNamesResponse>(
      detApi.StreamingInternal.metricNames(
        args.experimentId,
        undefined,
        { signal: canceler.signal },
      ),
      (event) => {
        if (!event) return;
        /*
         * The metrics endpoint can intermittently send empty lists,
         * so we keep track of what we have seen on our end and
         * only add new metrics we have not seen to the list.
         */
        (event.trainingMetrics || []).forEach((metric) => trainingMetricsMap[metric] = true);
        (event.validationMetrics || []).forEach((metric) => validationMetricsMap[metric] = true);
        const newTrainingMetrics = Object.keys(trainingMetricsMap).sort(alphaNumericSorter);
        const newValidationMetrics = Object.keys(validationMetricsMap).sort(alphaNumericSorter);
        const newMetrics = [
          ...newValidationMetrics.map((name) => ({ name, type: MetricType.Validation })),
          ...newTrainingMetrics.map((name) => ({ name, type: MetricType.Training })),
        ];
        if (newMetrics.length !== args.metricNames.length) {
          args.setMetricNames(newMetrics);
        }
      },
    ).catch(args.errorHandler);
    return () => canceler.abort();
  }, [ args ]);
};

export default useMetricNames;
