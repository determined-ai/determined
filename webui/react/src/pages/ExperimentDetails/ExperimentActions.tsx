import { Button, Popconfirm, Space } from 'antd';
import React, { useCallback, useState } from 'react';

import { ConditionalButton } from 'components/types';
import { openCommand } from 'routes/utils';
import {
  archiveExperiment, createTensorboard, killExperiment, setExperimentState,
} from 'services/api';
import { ExperimentDetails, RunState, TBSourceType } from 'types';
import { cancellableRunStates, killableRunStates, terminalRunStates } from 'utils/types';

export enum Action {
  Activate = 'Activate',
  Archive = 'Archive',
  Cancel = 'Cancel',
  Kill = 'Kill',
  Fork = 'Fork',
  Pause = 'Pause',
  Tensorboard = 'Tensorboard',
}

interface Props {
  experiment: ExperimentDetails;
  onClick: {
    [key in Action]?: () => void;
  };
  onSettled: () => void; // A callback to trigger after an action is done.
}

type ButtonLoadingStates = Record<Action, boolean>;

/*
  * We use `numSteps` or `totalBatchesProcessed` as a
  * proxy to trials that definietly have some metric.
  */
const experimentWillNeverHaveData = (experiment: ExperimentDetails): boolean => {
  const isTerminal = terminalRunStates.has(experiment.state);
  const trialsWithSomeMetric = experiment.trials.filter(trial => {
    return trial.numSteps > 1 || trial.totalBatchesProcessed > 0;
  });
  return isTerminal && trialsWithSomeMetric.length === 0;
};

const ExperimentActions: React.FC<Props> = ({ experiment, onClick, onSettled }: Props) => {
  const [ buttonStates, setButtonStates ] = useState<ButtonLoadingStates>({
    Activate: false,
    Archive: false,
    Cancel: false,
    Fork: false,
    Kill: false,
    Pause: false,
    Tensorboard: false,
  });

  const handleArchive = useCallback((archive: boolean) => async (): Promise<void> => {
    setButtonStates(state => ({ ...state, archive }));
    try {
      await archiveExperiment(experiment.id, archive);
      onSettled();
    } finally {
      setButtonStates(state => ({ ...state, archive: false }));
    }
  }, [ experiment.id, onSettled ]);

  const handleKill = useCallback(async () => {
    setButtonStates(state => ({ ...state, kill: true }));
    try {
      await killExperiment({ experimentId: experiment.id });
      onSettled();
    } finally {
      setButtonStates(state => ({ ...state, kill: false }));
    }
  }, [ experiment.id, onSettled ]);

  const handleCreateTensorboard = useCallback(async () => {
    setButtonStates(state => ({ ...state, tensorboard: true }));
    try {
      const tensorboard = await createTensorboard({
        ids: [ experiment.id ],
        type: TBSourceType.Experiment,
      });
      openCommand(tensorboard);
      onSettled();
    } finally {
      setButtonStates(state => ({ ...state, tensorboard: false }));
    }
  }, [ experiment.id, onSettled ]);

  const handleStateChange = useCallback((targetState: RunState) => async (): Promise<void> => {
    setButtonStates(state => ({ ...state, [targetState]: true }));
    try {
      await setExperimentState({ experimentId: experiment.id, state: targetState });
      onSettled();
    } finally {
      setButtonStates(state => ({ ...state, [targetState]: false }));
    }
  }, [ experiment.id, onSettled ]);

  const actionButtons: ConditionalButton<ExperimentDetails>[] = [
    {
      button: <Popconfirm
        cancelText="No"
        key="kill"
        okText="Yes"
        title="Are you sure you want to kill the experiment?"
        onConfirm={handleKill}>
        <Button danger loading={buttonStates.Kill} type="primary">Kill</Button>
      </Popconfirm>,
      showIf: (exp): boolean => killableRunStates.includes(exp.state),
    },
    {
      button: <Popconfirm
        cancelText="No"
        key="cancel"
        okText="Yes"
        title="Are you sure you want to cancel the experiment?"
        onConfirm={handleStateChange(RunState.StoppingCanceled)}>
        <Button danger loading={buttonStates.Cancel}>Cancel</Button>
      </Popconfirm>,
      showIf: (exp): boolean => cancellableRunStates.includes(exp.state),
    },
    {
      button: <Button
        key="pause"
        loading={buttonStates.Pause}
        onClick={handleStateChange(RunState.Paused)}>Pause</Button>,
      showIf: (exp): boolean => exp.state === RunState.Active,
    },
    {
      button: <Button
        key="activate"
        loading={buttonStates.Activate}
        type="primary"
        onClick={handleStateChange(RunState.Active)}>Activate</Button>,
      showIf: (exp): boolean => exp.state === RunState.Paused,
    },
    { button: <Button key="fork" onClick={onClick[Action.Fork]}>Fork</Button> },
    {
      button: <Button
        key="tensorboard"
        loading={buttonStates.Tensorboard}
        onClick={handleCreateTensorboard}>View in Tensorboard</Button>,
      showIf: (exp): boolean => !experimentWillNeverHaveData(exp),
    },
    {
      button: <Button
        key="archive"
        loading={buttonStates.Archive}
        onClick={handleArchive(true)}>Archive</Button>,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && !exp.archived,
    },
    {
      button: <Button
        key="unarchive"
        loading={buttonStates.Archive}
        onClick={handleArchive(false)}>Unarchive</Button>,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && exp.archived,
    },
  ];

  return (
    <Space size="small">
      {actionButtons
        .filter(ab => !ab.showIf || ab.showIf(experiment as ExperimentDetails))
        .map(ab => ab.button)
      }
    </Space>
  );
};

export default ExperimentActions;
