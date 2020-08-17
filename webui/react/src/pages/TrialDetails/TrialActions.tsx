import { Button, Space } from 'antd';
import React, { useCallback, useState } from 'react';

import Link from 'components/Link';
import { ConditionalButton } from 'components/types';
import { createTensorboard } from 'services/api';
import { RunState, TBSourceType, TrialDetails } from 'types';
import { openCommand } from 'utils/routes';
import { terminalRunStates } from 'utils/types';

export enum Action {
  Continue = 'Continue',
  Logs = 'Logs',
  Tensorboard = 'Tensorboard',
}

interface Props {
  trial: TrialDetails;
  onSettled: () => void; // A callback to trigger after an action is done.
  onClick: (action: Action) => (() => void);
}

type ButtonLoadingStates = Record<Action, boolean>;

const TrialActions: React.FC<Props> = ({ trial, onClick, onSettled: updateFn }: Props) => {

  const [ buttonStates, setButtonStates ] = useState<ButtonLoadingStates>({
    Continue: false,
    Logs: false,
    Tensorboard: false,
  });

  const handleCreateTensorboard = useCallback(() => {
    setButtonStates(state => ({ ...state, tensorboard: true }));
    createTensorboard({ ids: [ trial.id ], type: TBSourceType.Trial })
      .then((tensorboard) => {
        openCommand(tensorboard);
        return updateFn();
      })
      .finally(() => setButtonStates(state => ({ ...state, tensorboard: false })));
  }, [ trial.id, updateFn ]);

  const trialWillNeverHaveData = (trial: TrialDetails): boolean => {
    const isTerminal = terminalRunStates.has(trial.state);
    const stepsWithSomeMetric = trial.steps.filter(step => step.state === RunState.Completed);
    return isTerminal && stepsWithSomeMetric.length === 0;
  };

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
    { button: <Button key={Action.Logs}>
      <Link path={`/det/trials/${trial.id}/logs`} popout>Logs</Link>
    </Button> },
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
