import React, { useCallback, useMemo } from 'react';

import useSettings from 'hooks/useSettings';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { ExperimentBase, MetricName, MetricType, TrialDetails } from 'types';
import { extractMetricNames } from 'utils/trial';

import TrialChart from './TrialChart';
import settingsConfig, { Settings } from './TrialDetailsOverview.settings';
import TrialDetailsWorkloads from './TrialDetailsWorkloads';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialDetailsOverview: React.FC<Props> = ({ experiment, trial }: Props) => {
  const storagePath = `trial-detail/experiment/${experiment.id}`;
  const {
    settings,
    updateSettings,
  } = useSettings<Settings>(settingsConfig, { storagePath });

  const { defaultMetrics, metricNames, metrics } = useMemo(() => {
    const validationMetric = experiment?.config?.searcher.metric;
    const metricNames = extractMetricNames(trial?.workloads || []);
    const defaultValidationMetric = metricNames.find(metricName => (
      metricName.name === validationMetric && metricName.type === MetricType.Validation
    ));
    const fallbackMetric = metricNames && metricNames.length !== 0 ? metricNames[0] : undefined;
    const defaultMetric = defaultValidationMetric || fallbackMetric;
    const defaultMetrics = defaultMetric ? [ defaultMetric ] : [];
    const settingMetrics: MetricName[] = (settings.metric || []).map(metric => {
      const splitMetric = metric.split('|');
      return { name: splitMetric[1], type: splitMetric[0] as MetricType };
    });
    const metrics = settingMetrics.length !== 0 ? settingMetrics : defaultMetrics;
    return { defaultMetrics, metricNames, metrics };
  }, [ experiment?.config?.searcher, settings.metric, trial?.workloads ]);

  const handleMetricChange = useCallback((value: MetricName[]) => {
    const newMetrics = value.map(metricName => `${metricName.type}|${metricName.name}`);
    updateSettings({ metric: newMetrics, tableOffset: 0 });
  }, [ updateSettings ]);

  return (
    <>
      <TrialInfoBox experiment={experiment} trial={trial} />
      <TrialChart
        defaultMetricNames={defaultMetrics}
        metricNames={metricNames}
        metrics={metrics}
        workloads={trial?.workloads}
        onMetricChange={handleMetricChange}
      />
      <TrialDetailsWorkloads
        defaultMetrics={defaultMetrics}
        experiment={experiment}
        metricNames={metricNames}
        metrics={metrics}
        settings={settings}
        trial={trial}
        updateSettings={updateSettings}
      />
    </>
  );
};

export default TrialDetailsOverview;
