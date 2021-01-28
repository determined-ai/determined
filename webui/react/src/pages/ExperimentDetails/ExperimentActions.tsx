import { Button, Popconfirm, Space } from 'antd';
import React, { useCallback, useState } from 'react';

import { ConditionalButton } from 'components/types';
import {
  activateExperiment, archiveExperiment, cancelExperiment, killExperiment,
  openOrCreateTensorboard, pauseExperiment, unarchiveExperiment,
} from 'services/api';
import { ExperimentBase, RunState } from 'types';
import { cancellableRunStates, killableRunStates, terminalRunStates } from 'utils/types';
import { openCommand } from 'wait';

export enum Action {
  Activate = 'Activate',
  Archive = 'Archive',
  Cancel = 'Cancel',
  Fork = 'Fork',
  Kill = 'Kill',
  Pause = 'Pause',
  Tensorboard = 'Tensorboard',
  Unarchive = 'Unarchive',
}

interface Props {
  experiment: ExperimentBase;
  onClick: {
    [key in Action]?: () => void;
  };
  onSettled: () => void; // A callback to trigger after an action is done.
}

type ButtonLoadingStates = Record<Action, boolean>;

const ExperimentActions: React.FC<Props> = ({ experiment, onClick, onSettled }: Props) => {
  const [ btnLoadingStates, setBtnLoadingStates ] = useState<ButtonLoadingStates>({
    Activate: false,
    Archive: false,
    Cancel: false,
    Fork: false,
    Kill: false,
    Pause: false,
    Tensorboard: false,
    Unarchive: false,
  });

  const handleArchive = useCallback(() => async (): Promise<void> => {
    setBtnLoadingStates(state => ({ ...state, Archive: true }));
    try {
      await archiveExperiment({ experimentId: experiment.id });
      onSettled();
    } finally {
      setBtnLoadingStates(state => ({ ...state, Archive: false }));
    }
  }, [ experiment.id, onSettled ]);

  const handleUnarchive = useCallback(() => async (): Promise<void> => {
    setBtnLoadingStates(state => ({ ...state, Unarchive: true }));
    try {
      await unarchiveExperiment({ experimentId: experiment.id });
      onSettled();
    } finally {
      setBtnLoadingStates(state => ({ ...state, Unarchive: false }));
    }
  }, [ experiment.id, onSettled ]);

  const handleKill = useCallback(async () => {
    setBtnLoadingStates(state => ({ ...state, Kill: true }));
    try {
      await killExperiment({ experimentId: experiment.id });
      onSettled();
    } finally {
      setBtnLoadingStates(state => ({ ...state, Kill: false }));
    }
  }, [ experiment.id, onSettled ]);

  const handleCreateTensorboard = useCallback(async () => {
    setBtnLoadingStates(state => ({ ...state, Tensorboard: true }));
    try {
      const tensorboard = await openOrCreateTensorboard({ experimentIds: [ experiment.id ] });
      openCommand(tensorboard);
      onSettled();
    } finally {
      setBtnLoadingStates(state => ({ ...state, Tensorboard: false }));
    }
  }, [ experiment.id, onSettled ]);

  const handleStateChange = useCallback((targetState: RunState) => async (): Promise<void> => {
    let action: Action;
    switch (targetState) {
      case RunState.Active:
        action = Action.Activate;
        break;
      case RunState.Paused:
        action = Action.Pause;
        break;
      case RunState.Canceled:
      case RunState.StoppingCanceled:
        action = Action.Cancel;
        break;
      default:
        // unsupported targetState.
        return;
    }
    setBtnLoadingStates(state => ({ ...state, [action]: true }));
    try {
      switch (targetState) {
        case RunState.StoppingCanceled:
          return await cancelExperiment({ experimentId: experiment.id });
        case RunState.Paused:
          return await pauseExperiment({ experimentId: experiment.id });
        case RunState.Active:
          return await activateExperiment({ experimentId: experiment.id });
      }
      onSettled();
    } finally {
      setBtnLoadingStates(state => ({ ...state, [action]: false }));
    }
  }, [ experiment.id, onSettled ]);

  const actionButtons: ConditionalButton<ExperimentBase>[] = [
    {
      button: <Popconfirm
        cancelText="No"
        key="kill"
        okText="Yes"
        title="Are you sure you want to kill the experiment?"
        onConfirm={handleKill}>
        <Button danger loading={btnLoadingStates.Kill} type="primary">Kill</Button>
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
        <Button danger loading={btnLoadingStates.Cancel}>Cancel</Button>
      </Popconfirm>,
      showIf: (exp): boolean => cancellableRunStates.includes(exp.state),
    },
    {
      button: <Button
        key="pause"
        loading={btnLoadingStates.Pause}
        onClick={handleStateChange(RunState.Paused)}>Pause</Button>,
      showIf: (exp): boolean => exp.state === RunState.Active,
    },
    {
      button: <Button
        key="activate"
        loading={btnLoadingStates.Activate}
        type="primary"
        onClick={handleStateChange(RunState.Active)}>Activate</Button>,
      showIf: (exp): boolean => exp.state === RunState.Paused,
    },
    { button: <Button key="fork" onClick={onClick[Action.Fork]}>Fork</Button> },
    {
      button: <Button
        key="tensorboard"
        loading={btnLoadingStates.Tensorboard}
        onClick={handleCreateTensorboard}>View in TensorBoard</Button>,
    },
    {
      button: <Button
        key="archive"
        loading={btnLoadingStates.Archive}
        onClick={handleArchive()}>Archive</Button>,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && !exp.archived,
    },
    {
      button: <Button
        key="unarchive"
        loading={btnLoadingStates.Unarchive}
        onClick={handleUnarchive()}>Unarchive</Button>,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && exp.archived,
    },
  ];

  return (
    <Space size="small">
      {actionButtons
        .filter(ab => !ab.showIf || ab.showIf(experiment as ExperimentBase))
        .map(ab => ab.button)
      }
    </Space>
  );
};

export default ExperimentActions;
