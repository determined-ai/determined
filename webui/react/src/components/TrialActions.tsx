import { Button, Popconfirm } from 'antd';
import React, { useCallback, useState } from 'react';

import { archiveTrial, killTrial, launchTensorboard, setTrialState,
} from 'services/api';
import { TrialDetails, RunState, TBSourceType } from 'types';
import { openCommand } from 'utils/routes';
import { cancellableRunStates, killableRunStates, terminalRunStates } from 'utils/types';

import css from './TrialActions.module.scss';

interface Props {
  trial: TrialDetails;
  onSettled: () => void; // A callback to trigger after an action is done.
}

enum Action {
  Activate = 'Activate',
  Archive = 'Archive',
  Cancel = 'Cancel',
  Kill = 'Kill',
  Pause = 'Pause',
  Tensorboard = 'Tensorboard',
}

type ButtonLoadingStates = Record<Action, boolean>;

const TrialActions: React.FC<Props> = ({ trial, onSettled: updateFn }: Props) => {

  const [ buttonStates, setButtonStates ] = useState<ButtonLoadingStates>({
    Activate: false,
    Archive: false,
    Cancel: false,
    Kill: false,
    Pause: false,
    Tensorboard: false,
  });

  const handleArchive = useCallback((archive: boolean) =>
    (): Promise<unknown> => {
      setButtonStates(state => ({ ...state, archive: true }));
      return archiveTrial(trial.id, archive)
        .then(updateFn)
        .finally(() => setButtonStates(state => ({ ...state, archive: false })));
    },
  [ trial.id, updateFn ]);

  const handleKill = useCallback(() => {
    setButtonStates(state => ({ ...state, kill: true }));
    killTrial({ trialId: trial.id })
      .then(updateFn)
      .finally(() => setButtonStates(state => ({ ...state, kill: false })));
  }, [ trial.id, updateFn ]);

  const handleLaunchTensorboard = useCallback(() => {
    setButtonStates(state => ({ ...state, tensorboard: true }));
    launchTensorboard({ ids: [ trial.id ], type: TBSourceType.Trial })
      .then((tensorboard) => {
        openCommand(tensorboard);
        return updateFn();
      })
      .finally(() => setButtonStates(state => ({ ...state, tensorboard: false })));
  }, [ trial.id, updateFn ]);

  const handleStateChange = useCallback((targetState: RunState) =>
    (): Promise<unknown> => {
      setButtonStates(state => ({ ...state, [targetState]: true }));
      return setTrialState({ trialId: trial.id, state: targetState })
        .then(updateFn)
        .finally(() => setButtonStates(state => ({ ...state, [targetState]: false })));
    }
  , [ trial.id, updateFn ]);

  interface ConditionalButton {
    btn: React.ReactNode;
    showIf?: (exp: TrialDetails) => boolean;
  }

  const trialWillNeverHaveData = (trial: TrialDetails): boolean => {
    const isTerminal = terminalRunStates.has(trial.state);
    // with lack of step state we can use numSteps as a proxy to trials that definietly have some
    // metric.
    const trialsWithSomeMetric = trial.trials.filter(trial => trial.numSteps > 1);
    return isTerminal && trialsWithSomeMetric.length === 0;
  };

  const actionButtons: ConditionalButton[] = [
    { btn: <Button disabled key="fork" type="primary">Fork</Button> },
    {
      btn: <Button key="archive" loading={buttonStates.Archive}
        type="primary" onClick={handleArchive(true)}>
    Archive</Button>,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && !exp.archived,
    },
    {
      btn: <Button key="unarchive" loading={buttonStates.Archive}
        type="primary" onClick={handleArchive(false)}>
    Unarchive</Button>,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && exp.archived,
    },
    {
      btn: <Button key="pause" loading={buttonStates.Pause}
        type="primary" onClick={handleStateChange(RunState.Paused)}>
    Pause</Button>,
      showIf: (exp): boolean => exp.state === RunState.Active,
    },
    {
      btn: <Button key="activate" loading={buttonStates.Activate}
        type="primary" onClick={handleStateChange(RunState.Active)}>
    Activate</Button>,
      showIf: (exp): boolean => exp.state === RunState.Paused,
    },
    {
      btn: <Popconfirm
        cancelText="No"
        key="cancel"
        okText="Yes"
        title="Are you sure you want to kill the trial?"
        onConfirm={handleStateChange(RunState.StoppingCanceled)}
      >
        <Button danger loading={buttonStates.Cancel} type="primary">Cancel</Button>
      </Popconfirm>,
      showIf: (exp): boolean => cancellableRunStates.includes(exp.state),
    },
    {
      btn: <Popconfirm
        cancelText="No"
        key="kill"
        okText="Yes"
        title="Are you sure you want to kill the trial?"
        onConfirm={handleKill}
      >
        <Button danger loading={buttonStates.Kill} type="primary">Kill</Button>
      </Popconfirm>,
      showIf: (exp): boolean => killableRunStates.includes(exp.state),
    },
    {
      btn: <Button key="tensorboard"
        loading={buttonStates.Tensorboard} type="primary" onClick={handleLaunchTensorboard}>
      Tensorboard</Button>,
      showIf: (exp): boolean => !trialWillNeverHaveData(exp),
    },
  ];

  return (
    <ul className={css.base}>
      {actionButtons
        .filter(ab => !ab.showIf || ab.showIf(trial as TrialDetails))
        .map(ab => ab.btn)
      }
    </ul>
  );

};

export default TrialActions;
