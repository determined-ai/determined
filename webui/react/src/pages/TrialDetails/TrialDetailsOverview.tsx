import Spinner from 'hew/Spinner';
import { Loadable } from 'hew/utils/loadable';
import React, { useCallback, useMemo } from 'react';

import Metadata from 'components/Metadata';
import { terminalRunStates } from 'constants/states';
import useFeature from 'hooks/useFeature';
import useMetricNames from 'hooks/useMetricNames';
import usePermissions from 'hooks/usePermissions';
import { useSettings } from 'hooks/useSettings';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { ExperimentBase, Metric, MetricType, RunState, TrialDetails } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { metricKeyToMetric, metricToKey } from 'utils/metric';
import { capitalize } from 'utils/string';

import TrialChart from './TrialChart';
import { Settings, settingsConfigForExperiment } from './TrialDetailsOverview.settings';
import TrialDetailsWorkloads from './TrialDetailsWorkloads';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

const TrialDetailsOverview: React.FC<Props> = ({ experiment, trial }: Props) => {
  const storagePath = `trial-detail/experiment/${experiment.id}`;
  const settingsConfig = useMemo(() => settingsConfigForExperiment(experiment.id), [experiment.id]);
  const { settings, updateSettings } = useSettings<Settings>(
    Object.assign(settingsConfig, { storagePath }),
  );
  const f_flat_runs = useFeature().isOn('flat_runs');
  const entityCopy = f_flat_runs ? 'search' : 'experiment';

  const showExperimentArtifacts = usePermissions().canViewExperimentArtifacts({
    workspace: { id: experiment.workspaceId },
  });

  const handleMetricNamesError = useCallback(
    (e: unknown) => {
      handleError(e, {
        publicMessage: `Failed to load metric names for ${entityCopy} ${experiment.id}.`,
        publicSubject: `${capitalize(entityCopy)} metric name stream failed.`,
        type: ErrorType.Api,
      });
    },
    [entityCopy, experiment.id],
  );

  const trialNonTerminal = !terminalRunStates.has(experiment.state ?? RunState.Error);

  const experimentIds = useMemo(() => [experiment.id], [experiment.id]);
  const loadableMetricNames = useMetricNames(
    experimentIds,
    handleMetricNamesError,
    trialNonTerminal,
  );

  const defaultMetricNames = useMemo(() => [], []);
  const metricNames = useMemo(() => {
    return Loadable.getOrElse(defaultMetricNames, loadableMetricNames);
  }, [defaultMetricNames, loadableMetricNames]);

  const { defaultMetrics, metrics } = useMemo(() => {
    const validationMetric = experiment?.config?.searcher.metric;
    const defaultValidationMetric = metricNames.find(
      (metricName) =>
        metricName.name === validationMetric && metricName.group === MetricType.Validation,
    );
    const fallbackMetric = metricNames[0];
    const defaultMetric = defaultValidationMetric || fallbackMetric;
    const defaultMetrics = defaultMetric ? [defaultMetric] : [];
    const settingMetrics: Metric[] = (settings.metric || []).map((metric) =>
      metricKeyToMetric(metric),
    );
    const metrics = settingMetrics.length !== 0 ? settingMetrics : defaultMetrics;
    return { defaultMetrics, metrics };
  }, [experiment?.config?.searcher, metricNames, settings.metric]);

  const handleMetricChange = useCallback(
    (value: Metric[]) => {
      const newMetrics = value.map((metric) => metricToKey(metric));
      updateSettings({ metric: newMetrics, tableOffset: 0 });
    },
    [updateSettings],
  );

  return (
    <>
      <TrialInfoBox experiment={experiment} trial={trial} />
      {showExperimentArtifacts ? (
        <>
          <TrialChart
            defaultMetricNames={defaultMetrics}
            metricNames={metricNames}
            metrics={metrics}
            trialId={trial?.id}
            trialTerminated={terminalRunStates.has(trial?.state ?? RunState.Active)}
            onMetricChange={handleMetricChange}
          />
          {settings ? (
            <TrialDetailsWorkloads
              defaultMetrics={defaultMetrics}
              experiment={experiment}
              metricNames={metricNames}
              metrics={metrics}
              settings={settings}
              trial={trial}
              updateSettings={updateSettings}
            />
          ) : (
            <Spinner spinning />
          )}
          {trial?.metadata !== undefined && <Metadata trial={trial} />}
        </>
      ) : null}
    </>
  );
};

export default TrialDetailsOverview;
