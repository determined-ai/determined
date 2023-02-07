import { useEffect, useState } from 'react';

import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import { V1MetricNamesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import { alphaNumericSorter } from 'shared/utils/sort';
import { Metric, MetricType } from 'types';

const useMetricNames = (experimentId?: number, errorHandler?: (e: unknown) => void): Metric[] => {
  const [metrics, setMetrics] = useState<Metric[]>([]);

  useEffect(() => {
    if (!experimentId) return;
    const canceler = new AbortController();
    const trainingMetricsMap: Record<string, boolean> = {};
    const validationMetricsMap: Record<string, boolean> = {};

    // We do not want to plot any x-axis metric values as y-axis data
    const xAxisMetrics = Object.values(XAxisDomain).map((v) => v.toLowerCase());

    readStream<V1MetricNamesResponse>(
      detApi.StreamingInternal.metricNames(experimentId, undefined, {
        signal: canceler.signal,
      }),
      (event: V1MetricNamesResponse) => {
        if (!event) return;
        /*
         * The metrics endpoint can intermittently send empty lists,
         * so we keep track of what we have seen on our end and
         * only add new metrics we have not seen to the list.
         */
        (event.trainingMetrics || []).filter((metric) => !xAxisMetrics.includes(metric)).forEach((metric) => (trainingMetricsMap[metric] = true));
        (event.validationMetrics || []).filter((metric) => !xAxisMetrics.includes(metric)).forEach((metric) => (validationMetricsMap[metric] = true));
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
