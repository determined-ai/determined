import React, { useMemo, useState } from 'react';

import Icon from 'components/Icon';
import PageHeaderFoldable, { Option } from 'components/PageHeaderFoldable';
import TrialHeaderLeft from 'pages/TrialDetails/Header/TrialHeaderLeft';
import { openOrCreateTensorboard } from 'services/api';
import { getStateColorCssVar } from 'themes';
import { ExperimentBase, RunState, TrialDetails } from 'types';
import { getWorkload, isMetricsWorkload } from 'utils/step';
import { terminalRunStates } from 'utils/types';
import { openCommand } from 'wait';

export enum Action {
  Continue = 'Continue',
  Tensorboard = 'Tensorboard',
}

export const trialWillNeverHaveData = (trial: TrialDetails): boolean => {
  const isTerminal = terminalRunStates.has(trial.state);
  const workloadsWithSomeMetric = trial.workloads
    .map(getWorkload)
    .filter(isMetricsWorkload)
    .filter(workload => workload.metrics && workload.state === RunState.Completed);
  return isTerminal && workloadsWithSomeMetric.length === 0;
};

interface Props {
  experiment: ExperimentBase;
  fetchTrialDetails: () => void;
  handleActionClick: (action: Action) => void;
  trial: TrialDetails;
}

const TrialDetailsHeader: React.FC<Props> = (
  { experiment, fetchTrialDetails, handleActionClick, trial }: Props,
) => {
  const [ isRunningTensorboard, setIsRunningTensorboard ] = useState<boolean>(false);

  const headerOptions = useMemo<Option[]>(() => {
    const options: Option[] = [];

    if (trial.bestAvailableCheckpoint !== undefined) {
      options.push({
        icon: <Icon name="fork" size="small" />,
        key: Action.Continue,
        label: 'Continue Trial',
        onClick: () => handleActionClick(Action.Continue),
      });
    } else {
      options.push({
        icon: <Icon name="fork" size="small" />,
        key: Action.Continue,
        label: 'Continue Trial',
        tooltip: 'No checkpoints found. Cannot continue trial',
      });
    }

    if (!trialWillNeverHaveData(trial)) {
      options.push({
        icon: <Icon name="tensorboard" size="small" />,
        isLoading: isRunningTensorboard,
        key: Action.Tensorboard,
        label: 'TensorBoard',
        onClick: async () => {
          setIsRunningTensorboard(true);
          const tensorboard = await openOrCreateTensorboard({ trialIds: [ trial.id ] });
          openCommand(tensorboard);
          await fetchTrialDetails();
          setIsRunningTensorboard(false);
        },
      });
    }

    return options;
  }, [
    fetchTrialDetails,
    handleActionClick,
    isRunningTensorboard,
    trial,
  ]);

  return (
    <PageHeaderFoldable
      leftContent={<TrialHeaderLeft experiment={experiment} trial={trial} />}
      options={headerOptions}
      style={{ backgroundColor: getStateColorCssVar(trial.state) }}
    />
  );
};

export default TrialDetailsHeader;
