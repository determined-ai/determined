import { Button, Space, Tooltip } from 'antd';
import React, { useCallback, useState } from 'react';

import Link from 'components/Link';
import { ConditionalButton } from 'components/types';
import { openOrCreateTensorboard } from 'services/api';
import { RunState, TrialDetails } from 'types';
import { getWorkload, isMetricsWorkload } from 'utils/step';
import { terminalRunStates } from 'utils/types';
import { openCommand } from 'wait';

export enum Action {
  Continue = 'Continue',
  Logs = 'Logs',
  Tensorboard = 'Tensorboard',
}

interface Props {
  onClick: (action: Action) => (() => void);
  onSettled: () => void; // A callback to trigger after an action is done.
  trial: TrialDetails;
}

type ButtonLoadingStates = Record<Action, boolean>;

const trialWillNeverHaveData = (trial: TrialDetails): boolean => {
  const isTerminal = terminalRunStates.has(trial.state);
  const workloadsWithSomeMetric = trial.workloads
    .map(getWorkload)
    .filter(isMetricsWorkload)
    .filter(workload => workload.metrics && workload.state === RunState.Completed);
  return isTerminal && workloadsWithSomeMetric.length === 0;
};

const TrialActions: React.FC<Props> = ({ trial, onClick, onSettled }: Props) => {
  const [ buttonStates, setButtonStates ] = useState<ButtonLoadingStates>({
    Continue: false,
    Logs: false,
    Tensorboard: false,
  });

  const handleCreateTensorboard = useCallback(async () => {
    setButtonStates(state => ({ ...state, tensorboard: true }));
    const tensorboard = await openOrCreateTensorboard({ trialIds: [ trial.id ] });
    openCommand(tensorboard);
    onSettled();
    setButtonStates(state => ({ ...state, tensorboard: false }));
  }, [ trial.id, onSettled ]);

  const actionButtons: ConditionalButton<TrialDetails>[] = [
    {
      button: (trial.bestAvailableCheckpoint !== undefined ? (
        <Button key={Action.Continue} onClick={onClick(Action.Continue)}>Continue Trial</Button>
      ) : (
        <Tooltip key={Action.Continue} title={'No checkpoints found. Cannot continue trial.'}>
          <Button disabled>Continue Trial</Button>
        </Tooltip>
      )),
    },
    {
      button: <Button
        key={Action.Tensorboard}
        loading={buttonStates.Tensorboard}
        onClick={handleCreateTensorboard}>View in TensorBoard</Button>,
      showIf: (aTrial): boolean => !trialWillNeverHaveData(aTrial),
    },
    {
      button: <Button key={Action.Logs}>
        <Link path={`/experiments/${trial.experimentId}/trials/${trial.id}/logs`}>Logs</Link>
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
