import { Button, Popconfirm } from 'antd';
import React, { useCallback, useState } from 'react';

import { archiveExperiment, killExperiment, launchTensorboard, setExperimentState,
} from 'services/api';
import { ExperimentDetails, RunState, TBSourceType } from 'types';
import { openCommand } from 'utils/routes';
import { cancellableRunStates, killableRunStates, terminalRunStates } from 'utils/types';

import css from './ExperimentActions.module.scss';

interface Props {
  experiment: ExperimentDetails;
  finally: () => void; // A callback to trigger after an action is done.
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

const ExperimentActions: React.FC<Props> = ({ experiment, finally: updateFn }: Props) => {

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
      return archiveExperiment(experiment.id, archive)
        .then(updateFn)
        .finally(() => setButtonStates(state => ({ ...state, archive: false })));
    },
  [ experiment.id, updateFn ]);

  const handleKill = useCallback(() => {
    setButtonStates(state => ({ ...state, kill: true }));
    killExperiment({ experimentId: experiment.id })
      .then(updateFn)
      .finally(() => setButtonStates(state => ({ ...state, kill: false })));
  }, [ experiment.id, updateFn ]);

  const handleLaunchTensorboard = useCallback(() => {
    setButtonStates(state => ({ ...state, tensorboard: true }));
    launchTensorboard({ ids: [ experiment.id ], type: TBSourceType.Experiment })
      .then((tensorboard) => {
        openCommand(tensorboard);
        return updateFn();
      })
      .finally(() => setButtonStates(state => ({ ...state, tensorboard: false })));
  }, [ experiment.id, updateFn ]);

  const handleStateChange = useCallback((targetState: RunState) =>
    (): Promise<unknown> => {
      setButtonStates(state => ({ ...state, [targetState]: true }));
      return setExperimentState({ experimentId: experiment.id, state: targetState })
        .then(updateFn)
        .finally(() => setButtonStates(state => ({ ...state, [targetState]: false })));
    }
  , [ experiment.id, updateFn ]);

  const archiveButton = <Button key="archive" loading={buttonStates.Archive}
    type="primary" onClick={handleArchive(true)}>
    Archive</Button>;
  const unarchiveButton = <Button key="unarchive" loading={buttonStates.Archive}
    type="primary" onClick={handleArchive(false)}>
    Unarchive</Button>;

  const forkButton = <Button disabled key="fork" type="primary">Fork</Button>;
  const pauseButton = <Button key="pause" loading={buttonStates.Pause}
    type="primary" onClick={handleStateChange(RunState.Paused)}>
    Pause</Button>;
  const activateButton = <Button key="activate" loading={buttonStates.Activate}
    type="primary" onClick={handleStateChange(RunState.Active)}>
    Activate</Button>;

  const cancelButton = <Popconfirm
    cancelText="No"
    key="cancel"
    okText="Yes"
    title="Are you sure you want to kill the experiment?"
    onConfirm={handleStateChange(RunState.StoppingCanceled)}
  >
    <Button danger loading={buttonStates.Cancel}
      type="primary">
    Cancel</Button>
  </Popconfirm>;

  const killButton = <Popconfirm
    cancelText="No"
    key="kill"
    okText="Yes"
    title="Are you sure you want to kill the experiment?"
    onConfirm={handleKill}
  >
    <Button danger loading={buttonStates.Kill} type="primary">Kill</Button>
  </Popconfirm>;

  const tensorboardButton = <Button key="tensorboard"
    loading={buttonStates.Tensorboard} type="primary" onClick={handleLaunchTensorboard}>
      Tensorboard</Button>;

  interface ConditionalButton {
    btn: React.ReactNode;
    showIf?: (exp: ExperimentDetails) => boolean;
  }

  const experimentWillNeverHaveData = (experiment: ExperimentDetails): boolean => {
    const isTerminal = terminalRunStates.has(experiment.state);
    // with lack of step state we can use numSteps as a proxy to trials that definietly have some
    // metric.
    const trialsWithSomeMetric = experiment.trials.filter(trial => trial.numSteps > 1);
    return isTerminal && trialsWithSomeMetric.length === 0;
  };

  const actionButtons: ConditionalButton[] = [
    { btn: forkButton },
    {
      btn: archiveButton,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && !exp.archived,
    },
    {
      btn: unarchiveButton,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && exp.archived,
    },
    {
      btn: pauseButton,
      showIf: (exp): boolean => exp.state === RunState.Active,
    },
    {
      btn: activateButton,
      showIf: (exp): boolean => exp.state === RunState.Paused,
    },
    {
      btn: cancelButton,
      showIf: (exp): boolean => cancellableRunStates.includes(exp.state),
    },
    {
      btn: killButton,
      showIf: (exp): boolean => killableRunStates.includes(exp.state),
    },
    {
      btn: tensorboardButton,
      showIf: (exp): boolean => !experimentWillNeverHaveData(exp),
    },
  ];

  return (
    <ul className={css.base}>
      {actionButtons
        .filter(ab => !ab.showIf || ab.showIf(experiment as ExperimentDetails))
        .map(ab => ab.btn)
      }
    </ul>
  );

};

export default ExperimentActions;
