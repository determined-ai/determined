import React, { useCallback } from 'react';

import { terminalRunStates } from 'constants/states';
import useMetricNames from 'hooks/useMetricNames';
import usePermissions from 'hooks/usePermissions';
import TrialInfoBox from 'pages/TrialDetails/TrialInfoBox';
import { ErrorType } from 'shared/utils/error';
import { ExperimentBase, MetricType, RunState, TrialDetails } from 'types';
import handleError from 'utils/error';

import TrialCharts from './TrialCharts';
import css from './TrialDetailsOverview.module.scss';

export interface Props {
  experiment: ExperimentBase;
  trial?: TrialDetails;
}

const TrialDetailsOverview: React.FC<Props> = ({ experiment, trial }: Props) => {
  const showExperimentArtifacts = usePermissions().canViewExperimentArtifacts({
    workspace: { id: experiment.workspaceId },
  });

  const handleMetricNamesError = useCallback(
    (e: unknown) => {
      handleError(e, {
        publicMessage: `Failed to load metric names for experiment ${experiment.id}.`,
        publicSubject: 'Experiment metric name stream failed.',
        type: ErrorType.Api,
      });
    },
    [experiment.id],
  );

  const validationMetric = experiment?.config?.searcher.metric;
  const metricNames = useMetricNames(experiment.id, handleMetricNamesError).sort((a, b) => {
    if (a.name === validationMetric && a.type === MetricType.Validation) {
      return -1;
    } else if (b.name === validationMetric && b.type === MetricType.Validation) {
      return 1;
    }
    return 0;
  });

  return (
    <div className={css.base}>
      <TrialInfoBox experiment={experiment} trial={trial} />
      {showExperimentArtifacts ? (
        <>
          <TrialCharts
            metricNames={metricNames}
            trialId={trial?.id}
            trialTerminated={terminalRunStates.has(trial?.state ?? RunState.Active)}
          />
        </>
      ) : null}
    </div>
  );
};

export default TrialDetailsOverview;
