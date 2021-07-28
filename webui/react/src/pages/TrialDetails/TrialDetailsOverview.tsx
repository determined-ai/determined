import React, { useCallback, useEffect, useMemo, useState } from 'react';

import useStorage from 'hooks/useStorage';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { parseUrl } from 'routes/utils';
import { ExperimentBase, MetricName, MetricType, TrialDetails } from 'types';
import { extractMetricNames } from 'utils/trial';

import TrialChart from './TrialChart';
import TrialDetailsWorkloads from './TrialDetailsWorkloads';

const URL_ALL = 'all';

const STORAGE_CHART_METRICS_KEY = 'metrics/chart';
const STORAGE_PATH = 'trial-detail';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialDetailsOverview: React.FC<Props> = ({ experiment, trial }: Props) => {
  const [ defaultMetrics, setDefaultMetrics ] = useState<MetricName[]>([]);
  const storageMetricsPath = experiment ? `experiments/${experiment.id}` : undefined;
  const storage = useStorage(STORAGE_PATH);
  const [ isUrlParsed, setIsUrlParsed ] = useState(false);

  const storageChartMetricsKey =
    storageMetricsPath && `${storageMetricsPath}/${STORAGE_CHART_METRICS_KEY}`;

  const validationMetric = useMemo(
    () => experiment?.config?.searcher.metric,
    [ experiment?.config?.searcher.metric ],
  );
  const metricNames = useMemo(() => extractMetricNames(
    trial?.workloads || [],
  ), [ trial?.workloads ]);
  const defaultMetric = useMemo(() => {
    return metricNames.find(metricName => (
      metricName.name === validationMetric && metricName.type === MetricType.Validation
    ));
  }, [ metricNames, validationMetric ]);
  const fallbackMetric = metricNames && metricNames.length !== 0 ? metricNames[0] : undefined;
  const initMetric = defaultMetric || fallbackMetric;
  const [ metrics, setMetrics ] = useState<MetricName[]>(
    storage.getWithDefault(storageChartMetricsKey || '', initMetric ? [ initMetric ] : []),
  );

  /*
   * When filters changes update the page URL.
   */
  useEffect(() => {
    if (!isUrlParsed) return;

    const searchParams = new URLSearchParams;
    const url = parseUrl(window.location.href);

    // metrics
    if (metrics && metrics.length > 0) {
      metrics.forEach(metric => searchParams.append('metric', metric.name + '|' + metric.type));
    } else {
      searchParams.append('state', URL_ALL);
    }

    window.history.pushState(
      {},
      '',
      url.origin + url.pathname + '?' + searchParams.toString(),
    );
  }, [ isUrlParsed, metrics ]);

  /*
     * On first load: if filters are specified in URL, override default.
     */
  useEffect(() => {
    if (isUrlParsed) return;

    // If search params are not set, we default to user preferences
    const url = parseUrl(window.location.href);
    if (url.search === '') {
      setIsUrlParsed(true);
      return;
    }

    const urlSearchParams = url.searchParams;

    // metrics
    const visibleMetrics = urlSearchParams.getAll('metric');
    let metrics = defaultMetrics;
    if (visibleMetrics != null) {
      metrics = (visibleMetrics.map(metric => {
        const splitMetric = metric.split('|');
        return { name: splitMetric[0], type: splitMetric[1] as MetricType };
      }));
    }
    setIsUrlParsed(true);
    setMetrics(metrics);
  }, [ defaultMetrics, isUrlParsed, metrics ]);

  const handleMetricChange = useCallback((value: MetricName[]) => {
    setMetrics(value);

    if (storageChartMetricsKey) storage.set(storageChartMetricsKey, value);
  }, [ storage, storageChartMetricsKey ]);

  // Default to selecting config search metric only.
  useEffect(() => {
    const searcherName = experiment.config?.searcher?.metric;
    const defaultMetric = metricNames.find(metricName => {
      return metricName.name === searcherName && metricName.type === MetricType.Validation;
    });
    const defaultMetrics = defaultMetric ? [ defaultMetric ] : [];
    setDefaultMetrics(defaultMetrics);
  }, [ experiment, metricNames, storage ]);

  return (
    <>
      <TrialInfoBox experiment={experiment} trial={trial} />
      <TrialChart
        defaultMetricNames={defaultMetrics}
        handleMetricChange={handleMetricChange}
        metricNames={metricNames}
        metrics={metrics}
        workloads={trial?.workloads} />
      <TrialDetailsWorkloads
        defaultMetrics={defaultMetrics}
        experiment={experiment}
        handleMetricChange={handleMetricChange}
        metricNames={metricNames}
        metrics={metrics}
        trial={trial} />
    </>
  );
};

export default TrialDetailsOverview;
