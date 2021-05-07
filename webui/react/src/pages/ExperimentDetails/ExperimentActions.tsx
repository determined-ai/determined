import { Button, Popconfirm, Space } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';
import { useHistory } from 'react-router';

import { ConditionalButton } from 'components/types';
import { paths } from 'routes/utils';
import {
  activateExperiment, archiveExperiment, cancelExperiment, deleteExperiment, killExperiment,
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

const ExperimentActions: React.FC<Props> = ({ experiment, onClick, onSettled }: Props) => {
  const [ isRunningActivate, setIsRunningActivate ] = useState<boolean>(false);
  const [ isRunningArchive, setIsRunningArchive ] = useState<boolean>(false);
  const [ isRunningCancel, setIsRunningCancel ] = useState<boolean>(false);
  const [ isRunningDelete, setIsRunningDelete ] = useState<boolean>(false);
  const [ isRunningKill, setIsRunningKill ] = useState<boolean>(false);
  const [ isRunningPause, setIsRunningPause ] = useState<boolean>(false);
  const [ isRunningTensorboard, setIsRunningTensorboard ] = useState<boolean>(false);
  const [ isRunningUnarchive, setIsRunningUnarchive ] = useState<boolean>(false);
  const history = useHistory();

  useEffect(() => {
    setIsRunningArchive(false);
    setIsRunningUnarchive(false);
  }, [ experiment.archived ]);

  useEffect(() => {
    setIsRunningActivate(false);
    setIsRunningCancel(false);
    setIsRunningKill(false);
    setIsRunningPause(false);
  }, [ experiment.state ]);

  const handleArchive = useCallback(() => async (): Promise<void> => {
    setIsRunningArchive(true);
    try {
      await archiveExperiment({ experimentId: experiment.id });
      onSettled();
    } catch (e) {
      setIsRunningArchive(false);
    }
  }, [ experiment.id, onSettled ]);

  const handleUnarchive = useCallback(() => async (): Promise<void> => {
    setIsRunningUnarchive(true);
    try {
      await unarchiveExperiment({ experimentId: experiment.id });
      onSettled();
    } catch (e) {
      setIsRunningUnarchive(false);
    }
  }, [ experiment.id, onSettled ]);

  const handleDelete = useCallback(async () => {
    setIsRunningDelete(true);
    try {
      await deleteExperiment({ experimentId: experiment.id });
      history.push(paths.experimentList());
    } catch (e) {
      setIsRunningDelete(false);
    }
  }, [ experiment.id, history ]);

  const handleKill = useCallback(async () => {
    setIsRunningKill(true);
    try {
      await killExperiment({ experimentId: experiment.id });
      onSettled();
    } catch (e) {
      setIsRunningKill(false);
    }
  }, [ experiment.id, onSettled ]);

  const handleCreateTensorboard = useCallback(async () => {
    setIsRunningTensorboard(true);
    try {
      const tensorboard = await openOrCreateTensorboard({ experimentIds: [ experiment.id ] });
      openCommand(tensorboard);
      onSettled();
      setIsRunningTensorboard(false);
    } catch (e) {
      setIsRunningTensorboard(false);
    }
  }, [ experiment.id, onSettled ]);

  const handleStateChange = useCallback((targetState: RunState) => async (): Promise<void> => {
    switch (targetState) {
      case RunState.Canceled:
      case RunState.StoppingCanceled:
        try {
          setIsRunningCancel(true);
          await cancelExperiment({ experimentId: experiment.id });
          onSettled();
        } catch (e) {
          setIsRunningCancel(false);
        }
        break;
      case RunState.Paused:
        try {
          setIsRunningPause(true);
          await pauseExperiment({ experimentId: experiment.id });
          onSettled();
        } catch (e) {
          setIsRunningPause(false);
        }
        break;
      case RunState.Active:
        try {
          setIsRunningActivate(true);
          await activateExperiment({ experimentId: experiment.id });
          onSettled();
        } catch (e) {
          setIsRunningActivate(false);
        }
        break;
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
        <Button danger loading={isRunningKill} type="primary">Kill</Button>
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
        <Button danger loading={isRunningCancel}>Cancel</Button>
      </Popconfirm>,
      showIf: (exp): boolean => cancellableRunStates.includes(exp.state),
    },
    {
      button: <Button
        key="pause"
        loading={isRunningPause}
        onClick={handleStateChange(RunState.Paused)}>Pause</Button>,
      showIf: (exp): boolean => exp.state === RunState.Active,
    },
    {
      button: <Button
        key="activate"
        loading={isRunningActivate}
        type="primary"
        onClick={handleStateChange(RunState.Active)}>Activate</Button>,
      showIf: (exp): boolean => exp.state === RunState.Paused,
    },
    { button: <Button key="fork" onClick={onClick[Action.Fork]}>Fork</Button> },
    {
      button: <Button
        key="tensorboard"
        loading={isRunningTensorboard}
        onClick={handleCreateTensorboard}>View in TensorBoard</Button>,
    },
    {
      button: <Button
        key="archive"
        loading={isRunningArchive}
        onClick={handleArchive()}>Archive</Button>,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && !exp.archived,
    },
    {
      button: <Button
        key="unarchive"
        loading={isRunningUnarchive}
        onClick={handleUnarchive()}>Unarchive</Button>,
      showIf: (exp): boolean => terminalRunStates.has(exp.state) && exp.archived,
    },
    {
      button: <Popconfirm
        cancelText="No"
        key="delete"
        okText="Yes"
        placement='topRight'
        title="Are you sure you want to delete the experiment?"
        onConfirm={handleDelete}>
        <Button danger loading={isRunningDelete}>Delete</Button>
      </Popconfirm>,
      showIf: (exp): boolean => terminalRunStates.has(exp.state),
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
