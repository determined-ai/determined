import _ from 'lodash';
import { useEffect, useState } from 'react';

import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import { V1ExpMetricNamesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import { Metric, MetricType } from 'types';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { metricKeyToMetric, metricToKey } from 'utils/metric';
import { metricSorter } from 'utils/metric';
import { alphaNumericSorter } from 'utils/sort';

import usePrevious from './usePrevious';
const useMetricNames = (
  experimentIds: number[],
  errorHandler?: (e: unknown) => void,
): Loadable<Metric[]> => {
  const [metrics, setMetrics] = useState<Loadable<Metric[]>>(NotLoaded);
  const [actualExpIds, setActualExpIds] = useState<number[]>([]);
  const previousExpIds = usePrevious(actualExpIds, []);
  useEffect(
    () => setActualExpIds((prev) => (_.isEqual(prev, experimentIds) ? prev : experimentIds)),
    [experimentIds],
  );

  useEffect(() => {
    if (actualExpIds.length === 0) {
      setMetrics(Loaded([]));
      return;
    }
    if (!_.isEqual(actualExpIds, previousExpIds)) setMetrics(NotLoaded);
    const canceler = new AbortController();

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
        const trainingMetrics = (event.trainingMetrics ?? [])
          .filter((metric) => !xAxisMetrics.includes(metric))
          .reduce((acc, cur) => acc.add(cur), new Set<string>());
        const validationMetrics = (event.validationMetrics ?? [])
          .filter((metric) => !xAxisMetrics.includes(metric))
          .reduce((acc, cur) => acc.add(cur), new Set<string>());
        const newTrainingMetrics = [...trainingMetrics].sort(alphaNumericSorter);
        const newValidationMetrics = [...validationMetrics].sort(alphaNumericSorter);
        const newMetrics = [
          ...newValidationMetrics.map((name) => ({ name, type: MetricType.Validation })),
          ...newTrainingMetrics.map((name) => ({ name, type: MetricType.Training })),
        ];
        setMetrics((prevMetrics) => {
          if (newMetrics.length === 0) {
            return Loadable.isLoaded(prevMetrics) ? prevMetrics : Loaded([]);
          }

          /*
           * Since we may intermittently receive a subset of all available
           * metrics or an empty list of metrics we must merge the new and
           * previous metrics to accurately determine if any new metrics have
           * not been seen before.
           */
          const previousMetrics = Loadable.getOrElse([], prevMetrics);

          const previousMetricsSet = Loadable.getOrElse([], prevMetrics).reduce(
            (acc, cur) => acc.add(metricToKey(cur)),
            new Set<string>(),
          );
          const updatedMetricsSet = [...newMetrics, ...previousMetrics].reduce(
            (acc, cur) => acc.add(metricToKey(cur)),
            new Set<string>(),
          );

          if (_.isEqual(previousMetricsSet, updatedMetricsSet)) return prevMetrics;

          return Loaded(
            Array.from(updatedMetricsSet)
              .map((metricKey) => metricKeyToMetric(metricKey))
              .filter((metric): metric is Metric => !!metric)
              .sort(metricSorter),
          );
        });
      },
      errorHandler,
    );
    return () => canceler.abort();
  }, [actualExpIds, previousExpIds, errorHandler]);
  return metrics;
};

export default useMetricNames;
