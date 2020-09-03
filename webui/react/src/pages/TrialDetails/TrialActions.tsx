import { Button, Space } from 'antd';
import React, { useCallback, useState } from 'react';

import Link from 'components/Link';
import { ConditionalButton } from 'components/types';
import { openCommand } from 'routes/utils';
import { createTensorboard } from 'services/api';
import { RunState, TBSourceType, TrialDetails } from 'types';
import { terminalRunStates } from 'utils/types';

export enum Action {
  Continue = 'Continue',
  Logs = 'Logs',
  Tensorboard = 'Tensorboard',
}

interface Props {
  trial: TrialDetails;
  onClick: (action: Action) => (() => void);
  onSettled: () => void; // A callback to trigger after an action is done.
}

type ButtonLoadingStates = Record<Action, boolean>;

const trialWillNeverHaveData = (trial: TrialDetails): boolean => {
  const isTerminal = terminalRunStates.has(trial.state);
  const stepsWithSomeMetric = trial.steps.filter(step => step.state === RunState.Completed);
  return isTerminal && stepsWithSomeMetric.length === 0;
};

const TrialActions: React.FC<Props> = ({ trial, onClick, onSettled }: Props) => {
  const [ buttonStates, setButtonStates ] = useState<ButtonLoadingStates>({
    Continue: false,
    Logs: false,
    Tensorboard: false,
  });

  const handleCreateTensorboard = useCallback(async () => {
    setButtonStates(state => ({ ...state, tensorboard: true }));
    const tensorboard = await createTensorboard({ ids: [ trial.id ], type: TBSourceType.Trial });
    openCommand(tensorboard);
    onSettled();
    setButtonStates(state => ({ ...state, tensorboard: false }));
  }, [ trial.id, onSettled ]);

  const actionButtons: ConditionalButton<TrialDetails>[] = [
    {
      button: <Button
        key={Action.Continue}
        onClick={onClick(Action.Continue)}>Continue Trial</Button>,
    },
    {
      button: <Button
        key={Action.Tensorboard}
        loading={buttonStates.Tensorboard}
        onClick={handleCreateTensorboard}>View in Tensorboard</Button>,
      showIf: (aTrial): boolean => !trialWillNeverHaveData(aTrial),
    },
    {
      button: <Button key={Action.Logs}>
        <Link path={`/det/trials/${trial.id}/logs`} popout>Logs</Link>
      </Button>,
    },
  ];

  return (
    <Space size="small">
      {actionButtons
        .filter(ab => !ab.showIf || ab.showIf(trial))
        .map(ab => ab.button)
      }
    </Space>
  );

};

export default TrialActions;
