import React, { useEffect, useMemo, useState } from 'react';

import useStorage from 'hooks/useStorage';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { ExperimentBase, MetricName, MetricType, TrialDetails } from 'types';
import { extractMetricNames } from 'utils/trial';

import TrialChart from './TrialChart';

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

  const storageChartMetricsKey =
    storageMetricsPath && `${storageMetricsPath}/${STORAGE_CHART_METRICS_KEY}`;

  const metricNames = useMemo(() => extractMetricNames(
    trial?.workloads || [],
  ), [ trial?.workloads ]);

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
        metricNames={metricNames}
        storageKey={storageChartMetricsKey}
        validationMetric={experiment?.config?.searcher.metric}
        workloads={trial?.workloads} />
    </>
  );
};

export default TrialDetailsOverview;
