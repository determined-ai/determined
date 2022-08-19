import React, { useCallback, useMemo, useState } from 'react';

import useMetricNames from 'hooks/useMetricNames';
import useSettings from 'hooks/useSettings';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { ErrorType } from 'shared/utils/error';
import { ExperimentBase, Metric, MetricType, RunState, TrialDetails } from 'types';
import handleError from 'utils/error';
import { extractMetrics } from 'utils/metric';

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

  const [ metrics, setMetrics ] = useState<Metric[]>([]);
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
    metrics,
    setMetrics,
  });

  const { defaultMetrics, selectedMetrics } = useMemo(() => {
    const validationMetric = experiment?.config?.searcher.metric;
    const defaultValidationMetric = metrics.find((metric) => (
      metric.name === validationMetric && metric.type === MetricType.Validation
    ));
    const fallbackMetric = metrics[0];
    const defaultMetric = defaultValidationMetric || fallbackMetric;
    const defaultMetrics = defaultMetric ? [ defaultMetric ] : [];
    const settingMetrics: Metric[] = (settings.metric || []).map((metric) => {
      const splitMetric = metric.split('|');
      return { name: splitMetric[1], type: splitMetric[0] as MetricType };
    });
    const selectedMetrics = settingMetrics.length !== 0 ? settingMetrics : defaultMetrics;
    return { defaultMetrics, selectedMetrics };
  }, [ experiment?.config?.searcher, metrics, settings.metric ]);

  const handleMetricChange = useCallback((value: Metric[]) => {
    const newMetrics = value.map((metric) => `${metric.type}|${metric.name}`);
    updateSettings({ metric: newMetrics, tableOffset: 0 });
  }, [ updateSettings ]);

  return (
    <div className={css.base}>
      <TrialInfoBox experiment={experiment} trial={trial} />
      <TrialChart
        defaultMetrics={defaultMetrics}
        metrics={metrics}
        selectedMetrics={selectedMetrics}
        trialId={trial?.id}
        trialTerminated={trial ?
          [ RunState.Completed, RunState.Errored ].includes(trial.state)
          : false}
        onMetricChange={handleMetricChange}
      />
      <TrialDetailsWorkloads
        defaultMetrics={defaultMetrics}
        experiment={experiment}
        selectedMetrics={metrics}
        settings={settings}
        trial={trial}
        updateSettings={updateSettings}
      />
    </div>
  );
};

export default TrialDetailsOverview;
