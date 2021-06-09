import React, { useMemo, useState } from 'react';

import Icon from 'components/Icon';
import PageHeaderFoldable, { Option } from 'components/PageHeaderFoldable';
import TrialHeaderLeft from 'pages/TrialDetails/Header/TrialHeaderLeft';
import { Action, trialWillNeverHaveData } from 'pages/TrialDetails/TrialActions';
import { openOrCreateTensorboard } from 'services/api';
import { getStateColorCssVar } from 'themes';
import { TrialDetails } from 'types';
import { openCommand } from 'wait';

interface Props {
  fetchTrialDetails: () => void;
  handleActionClick: (action: Action) => void;
  trial: TrialDetails;
}

const TrialDetailsHeader: React.FC<Props> = (
  { fetchTrialDetails, handleActionClick, trial }: Props,
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
      leftContent={<TrialHeaderLeft trial={trial} />}
      options={headerOptions}
      style={{ backgroundColor: getStateColorCssVar(trial.state) }}
    />
  );
};

export default TrialDetailsHeader;
