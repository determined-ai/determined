import { useEffect, useState } from 'react';

import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import { V1ExpMetricNamesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import { isEqual } from 'shared/utils/data';
import { alphaNumericSorter } from 'shared/utils/sort';
import { Metric, MetricType } from 'types';

const useMetricNames = (experimentIds: number[], errorHandler?: (e: unknown) => void): Metric[] => {
  const [metrics, setMetrics] = useState<Metric[]>([]);
  const [actualExpIds, setActualExpIds] = useState<number[]>([]);

  useEffect(
    () => setActualExpIds((prev) => (isEqual(prev, experimentIds) ? prev : experimentIds)),
    [experimentIds],
  );

  useEffect(() => {
    if (actualExpIds.length === 0) {
      setMetrics([]);
      return;
    }
    const canceler = new AbortController();
    const trainingMetricsMap: Record<string, boolean> = {};
    const validationMetricsMap: Record<string, boolean> = {};

    // We do not want to plot any x-axis metric values as y-axis data
    const xAxisMetrics = Object.values(XAxisDomain).map((v) => v.toLowerCase());

    readStream<V1ExpMetricNamesResponse>(
      detApi.StreamingInternal.expMetricNames(actualExpIds, undefined, {
        signal: canceler.signal,
      }),
      (event: V1ExpMetricNamesResponse) => {
        if (!event) return;
        /*
         * The metrics endpoint can intermittently send empty lists,
         * so we keep track of what we have seen on our end and
         * only add new metrics we have not seen to the list.
         */
        (event.trainingMetrics || [])
          .filter((metric) => !xAxisMetrics.includes(metric))
          .forEach((metric) => (trainingMetricsMap[metric] = true));
        (event.validationMetrics || [])
          .filter((metric) => !xAxisMetrics.includes(metric))
          .forEach((metric) => (validationMetricsMap[metric] = true));
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
  }, [actualExpIds, errorHandler]);
  return metrics;
};

export default useMetricNames;
