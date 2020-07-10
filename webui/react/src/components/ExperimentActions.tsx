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

const ExperimentActions: React.FC<Props> = ({ experiment, finally: updateFn }: Props) => {

  interface ButtonLoadingStates {
    kill: boolean;
    [RunState.StoppingCanceled]: boolean;
    archive: boolean;
    [RunState.Paused]: boolean;
    [RunState.Active]: boolean;
    tsb: boolean; // tensorboard
  }

  const [ buttonStates, setButtonStates ] = useState<ButtonLoadingStates>({
    [RunState.StoppingCanceled]: false,
    archive: false,
    [RunState.Active]: false,
    kill: false,
    [RunState.Paused]: false,
    tsb: false,
  });

  const killExperimentCB = useCallback(() => {
    setButtonStates(state => ({ ...state, kill: true }));
    killExperiment({ experimentId: experiment.id })
      .then(updateFn)
      .finally(() => setButtonStates(state => ({ ...state, kill: false })));
  }, [ experiment.id, updateFn ]);

  const archiveCB = useCallback((archive: boolean) =>
    (): Promise<unknown> => {
      setButtonStates(state => ({ ...state, archive: true }));
      return archiveExperiment(experiment.id, archive)
        .then(updateFn)
        .finally(() => setButtonStates(state => ({ ...state, archive: false })));
    },
  [ experiment.id, updateFn ]);

  const launchTensorboardCB = useCallback(() => {
    setButtonStates(state => ({ ...state, tsb: true }));
    launchTensorboard({ ids: [ experiment.id ], type: TBSourceType.Experiment })
      .then((tensorboard) => {
        openCommand(tensorboard);
        return updateFn();
      })
      .finally(() => setButtonStates(state => ({ ...state, tsb: false })));
  }, [ experiment.id, updateFn ]);

  const requestExpStateCB = useCallback((targetState: RunState) =>
    (): Promise<unknown> => {
      setButtonStates(state => ({ ...state, [targetState]: true }));
      return setExperimentState({ experimentId: experiment.id, state: targetState })
        .then(updateFn)
        .finally(() => setButtonStates(state => ({ ...state, [targetState]: false })));
    }
  , [ experiment.id, updateFn ]);

  const archiveButton = <Button key="archive" loading={buttonStates.archive}
    type="primary" onClick={archiveCB(true)}>
    Archive</Button>;
  const unarchiveButton = <Button key="unarchive" loading={buttonStates.archive}
    type="primary" onClick={archiveCB(false)}>
    Unarchive</Button>;

  const forkButton = <Button disabled key="fork" type="primary">Fork</Button>;
  const pauseButton = <Button key="pause" loading={buttonStates[RunState.Paused]}
    type="primary" onClick={requestExpStateCB(RunState.Paused)}>
    Pause</Button>;
  const activateButton = <Button key="activate" loading={buttonStates[RunState.Active]}
    type="primary" onClick={requestExpStateCB(RunState.Active)}>
    Activate</Button>;

  const cancelButton = <Popconfirm
    cancelText="No"
    key="cancel"
    okText="Yes"
    title="Are you sure you want to kill the experiment?"
    onConfirm={requestExpStateCB(RunState.StoppingCanceled)}
  >
    <Button danger loading={buttonStates[RunState.StoppingCanceled]}
      type="primary">
    Cancel</Button>
  </Popconfirm>;

  const killButton = <Popconfirm
    cancelText="No"
    key="kill"
    okText="Yes"
    title="Are you sure you want to kill the experiment?"
    onConfirm={killExperimentCB}
  >
    <Button danger loading={buttonStates.kill} type="primary">Kill</Button>
  </Popconfirm>;

  const tsbButton = <Button key="tensorboard"
    loading={buttonStates.tsb} type="primary" onClick={launchTensorboardCB}>
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
      btn: tsbButton,
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
