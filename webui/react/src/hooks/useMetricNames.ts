import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import _ from 'lodash';
import { useEffect, useMemo, useRef, useState } from 'react';

import { V1ExpMetricNamesResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { readStream } from 'services/utils';
import { Metric, XAxisDomain } from 'types';
import { metricKeyToMetric, metricSorter, metricToKey } from 'utils/metric';

const useMetricNames = (
  experimentIds: number[],
  errorHandler?: (e: unknown) => void,
  quickPoll?: boolean,
): Loadable<Metric[]> => {
  const [metrics, setMetrics] = useState<Loadable<Metric[]>>(NotLoaded);
  // Do not replace with usePrevious here -- it restarts the stream erroneously;
  const idsRef = useRef(experimentIds);
  const curExperimentIds = useMemo(() => {
    return _.isEqual(experimentIds, idsRef.current) ? idsRef.current : experimentIds;
  }, [experimentIds]);

  useEffect(() => {
    const previousExpIds = idsRef.current;
    if (curExperimentIds.length === 0) {
      setMetrics(Loaded([]));
      return;
    }
    if (curExperimentIds !== previousExpIds) setMetrics(NotLoaded);

    const canceler = new AbortController();

    // We do not want to plot any x-axis metric values as y-axis data
    const xAxisMetrics = Object.values(XAxisDomain).map((v) => v.toLowerCase());

    readStream<V1ExpMetricNamesResponse>(
      detApi.StreamingInternal.expMetricNames(curExperimentIds, quickPoll ? 5 : undefined, {
        signal: canceler.signal,
      }),
      (event: V1ExpMetricNamesResponse) => {
        if (!event) return;
        /**
         * The metrics endpoint can intermittently send empty lists,
         * so we keep track of what we have seen on our end and
         * only add new metrics we have not seen to the list.
         */
        setMetrics((prevMetrics) => {
          const newMetrics = (event.metricNames ?? []).filter(
            (metric) => !xAxisMetrics.includes(metric.name),
          );

          if (newMetrics.length === 0) {
            return Loadable.isLoaded(prevMetrics) ? prevMetrics : Loaded([]);
          }

          /**
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
              .sort(metricSorter),
          );
        });
      },
      errorHandler,
    );
    return () => canceler.abort();
  }, [curExperimentIds, errorHandler, quickPoll]);
  useEffect(() => {
    idsRef.current = experimentIds;
  });

  return metrics;
};

export default useMetricNames;
