import React, { useCallback, useMemo, useState } from 'react';

import useMetricNames from 'hooks/useMetricNames';
import useSettings from 'hooks/useSettings';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { ErrorType } from 'shared/utils/error';
import { ExperimentBase, MetricName, MetricType, RunState, TrialDetails } from 'types';
import handleError from 'utils/error';

import TrialChart from './TrialChart';
import css from './TrialDetailsOverview.module.scss';
import settingsConfig, { Settings } from './TrialDetailsOverview.settings';
import TrialDetailsWorkloads from './TrialDetailsWorkloads';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

const TrialDetailsOverview: React.FC<Props> = ({ experiment, trial }: Props) => {
  const storagePath = `trial-detail/experiment/${experiment.id}`;
  const {
    settings,
    updateSettings,
  } = useSettings<Settings>(settingsConfig, { storagePath });

  const [ metricNames, setMetricNames ] = useState<MetricName[]>([]);
  useMetricNames({
    errorHandler: () => {
      try {
        handleError({
          publicMessage: `Failed to load metric names for experiment ${experiment.id}.`,
          publicSubject: 'Experiment metric name stream failed.',
          type: ErrorType.Api,
        });
      } catch (e) {
        // already handleError
      }
    },
    experimentId: experiment.id,
    metricNames,
    setMetricNames,
  });

  const { defaultMetrics, metrics } = useMemo(() => {
    const validationMetric = experiment?.config?.searcher.metric;
    const defaultValidationMetric = metricNames.find((metricName) => (
      metricName.name === validationMetric && metricName.type === MetricType.Validation
    ));
    const fallbackMetric = metricNames[0];
    const defaultMetric = defaultValidationMetric || fallbackMetric;
    const defaultMetrics = defaultMetric ? [ defaultMetric ] : [];
    const settingMetrics: MetricName[] = (settings.metric || []).map((metric) => {
      const splitMetric = metric.split('|');
      return { name: splitMetric[1], type: splitMetric[0] as MetricType };
    });
    const metrics = settingMetrics.length !== 0 ? settingMetrics : defaultMetrics;
    return { defaultMetrics, metrics };
  }, [ experiment?.config?.searcher, metricNames, settings.metric ]);

  const handleMetricChange = useCallback((value: MetricName[]) => {
    const newMetrics = value.map((metricName) => `${metricName.type}|${metricName.name}`);
    updateSettings({ metric: newMetrics, tableOffset: 0 });
  }, [ updateSettings ]);

  return (
    <div className={css.base}>
      <TrialInfoBox experiment={experiment} trial={trial} />
      <TrialChart
        defaultMetricNames={defaultMetrics}
        metricNames={metricNames}
        metrics={metrics}
        trialId={trial?.id}
        trialTerminated={trial ?
          [ RunState.Completed, RunState.Errored ].includes(trial.state)
          : false}
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
    </div>
  );
};

export default TrialDetailsOverview;
