import React, { useMemo, useState } from 'react';

import Icon from 'components/Icon';
import PageHeaderFoldable, { Option } from 'components/PageHeaderFoldable';
import TrialHeaderLeft from 'pages/TrialDetails/Header/TrialHeaderLeft';
import { openOrCreateTensorboard } from 'services/api';
import { getStateColorCssVar } from 'themes';
import { ExperimentAction as Action, ExperimentBase, TrialDetails } from 'types';
import { terminalRunStates } from 'utils/types';
import { getWorkload, isMetricsWorkload } from 'utils/workload';
import { openCommand } from 'wait';

export const trialWillNeverHaveData = (trial: TrialDetails): boolean => {
  const isTerminal = terminalRunStates.has(trial.state);
  const workloadsWithSomeMetric = trial.workloads
    .map(getWorkload)
    .filter(wl => isMetricsWorkload(wl) && wl.metrics);
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

    if (!trialWillNeverHaveData(trial)) {
      options.push({
        icon: <Icon name="tensorboard" size="small" />,
        isLoading: isRunningTensorboard,
        key: Action.OpenTensorBoard,
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

    if (trial.bestAvailableCheckpoint !== undefined) {
      options.push({
        icon: <Icon name="fork" size="small" />,
        key: Action.ContinueTrial,
        label: 'Continue Trial',
        onClick: () => handleActionClick(Action.ContinueTrial),
      });
    } else {
      options.push({
        icon: <Icon name="fork" size="small" />,
        key: Action.ContinueTrial,
        label: 'Continue Trial',
        tooltip: 'No checkpoints found. Cannot continue trial',
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
