import { Button, Space, Tooltip } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import Link from 'components/Link';
import { ConditionalButton } from 'components/types';
import { openOrCreateTensorboard } from 'services/api';
import { RunState, TBSourceType, TrialDetails2, TrialItem } from 'types';
import { getWorkload, isMetricsWorkload } from 'utils/step';
import { terminalRunStates } from 'utils/types';
import { openCommand } from 'wait';

export enum Action {
  Continue = 'Continue',
  Logs = 'Logs',
  Tensorboard = 'Tensorboard',
}

interface Props {
  trial: TrialDetails2;
  trials: TrialItem[],
  onClick: (action: Action) => (() => void);
  onSettled: () => void; // A callback to trigger after an action is done.
}

type ButtonLoadingStates = Record<Action, boolean>;

const trialWillNeverHaveData = (trial: TrialDetails2): boolean => {
  const isTerminal = terminalRunStates.has(trial.state);
  const workloadsWithSomeMetric = trial.workloads
    .map(getWorkload)
    .filter(isMetricsWorkload)
    .filter(workload => workload.metrics && workload.state === RunState.Completed);
  return isTerminal && workloadsWithSomeMetric.length === 0;
};

const TrialActions: React.FC<Props> = ({ trial, trials, onClick, onSettled }: Props) => {
  const [ buttonStates, setButtonStates ] = useState<ButtonLoadingStates>({
    Continue: false,
    Logs: false,
    Tensorboard: false,
  });

  const handleCreateTensorboard = useCallback(async () => {
    setButtonStates(state => ({ ...state, tensorboard: true }));
    const tensorboard = await openOrCreateTensorboard({
      ids: [ trial.id ],
      type: TBSourceType.Trial,
    });
    openCommand(tensorboard);
    onSettled();
    setButtonStates(state => ({ ...state, tensorboard: false }));
  }, [ trial.id, onSettled ]);

  const trialCompletedCheckpointSum = useMemo(() => {
    return trials.reduce((acc, trial) => acc + trial.numCompletedCheckpoints, 0);
  }, [ trials ]);

  const actionButtons: ConditionalButton<TrialDetails2>[] = [
    {
      button: (trialCompletedCheckpointSum > 0 ? (
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
