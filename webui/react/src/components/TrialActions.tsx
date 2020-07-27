import { Button } from 'antd';
import React, { useCallback, useState } from 'react';

import Link from 'components/Link';
import { ConditionalButton } from 'components/types';
import { launchTensorboard } from 'services/api';
import { RunState, TBSourceType, TrialDetails } from 'types';
import { openCommand } from 'utils/routes';
import { terminalRunStates } from 'utils/types';

import css from './TrialActions.module.scss';

interface Props {
  trial: TrialDetails;
  onSettled: () => void; // A callback to trigger after an action is done.
}

enum Action {
  Continue = 'Continue',
  Logs = 'Logs',
  Tensorboard = 'Tensorboard',
}

type ButtonLoadingStates = Record<Action, boolean>;

const TrialActions: React.FC<Props> = ({ trial, onSettled: updateFn }: Props) => {

  const [ buttonStates, setButtonStates ] = useState<ButtonLoadingStates>({
    Continue: false,
    Logs: false,
    Tensorboard: false,
  });

  const handleLaunchTensorboard = useCallback(() => {
    setButtonStates(state => ({ ...state, tensorboard: true }));
    launchTensorboard({ ids: [ trial.id ], type: TBSourceType.Trial })
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
    { button: <Button disabled key={Action.Continue} type="primary">Continue Trial</Button> },
    { button: <Button key={Action.Logs} type="primary">
      <Link path={`/det/trials/${trial.id}/logs`} popout>Logs</Link>
    </Button> },
    {
      button: <Button key={Action.Tensorboard}
        loading={buttonStates.Tensorboard} type="primary" onClick={handleLaunchTensorboard}>
      Tensorboard</Button>,
      showIf: (aTrial): boolean => !trialWillNeverHaveData(aTrial),
    },
  ];

  return (
    <ul className={css.base}>
      {actionButtons
        .filter(ab => !ab.showIf || ab.showIf(trial))
        .map(ab => ab.button)
      }
    </ul>
  );

};

export default TrialActions;
