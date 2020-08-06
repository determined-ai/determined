import { Dropdown, Menu } from 'antd';
import { ClickParam } from 'antd/es/menu';
import React from 'react';

import Icon from 'components/Icon';
import Experiments from 'contexts/Experiments';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { setupUrlForDev } from 'routes';
import { archiveExperiment, createTensorboard, killTask, setExperimentState } from 'services/api';
import { AnyTask, CommandTask, Experiment, RunState, TBSourceType } from 'types';
import { openBlank, openCommand } from 'utils/routes';
import { capitalize } from 'utils/string';
import { isExperimentTask } from 'utils/task';
import { cancellableRunStates, isTaskKillable, terminalRunStates } from 'utils/types';

import css from './TaskActionDropdown.module.scss';

interface Props {
  task: AnyTask;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const TaskActionDropdown: React.FC<Props> = ({ task }: Props) => {
  const isExperiment = isExperimentTask(task);
  const isArchivable = isExperiment && terminalRunStates.has(task.state as RunState);
  const isKillable = isTaskKillable(task);
  const isPausable = isExperiment
    && task.state === RunState.Active;
  const isResumable = isExperiment
    && task.state === RunState.Paused;
  const isCancelable = isExperiment
    && cancellableRunStates.includes(task.state as RunState);

  const experimentsResponse = Experiments.useStateContext();
  const setExperiments = Experiments.useActionContext();

  // update the local state of a single experiment.
  // TODO refactor to contexts.
  const updateExperimentLocally = (updater: (arg0: Experiment) => Experiment): void => {
    if (experimentsResponse.data) {
      const experiments = experimentsResponse.data
        .map(exp => exp.id.toString() === task.id ? updater(exp) : exp);
      setExperiments({
        type: Experiments.ActionType.Set,
        value: { ...experimentsResponse, data: experiments },
      });
    }
  };

  const handleMenuClick = async (params: ClickParam): Promise<void> => {
    params.domEvent.stopPropagation();
    try {
      switch (params.key) { // Cases should match menu items.
        case 'activate':
          await setExperimentState({
            experimentId: parseInt(task.id),
            state: RunState.Active,
          });
          updateExperimentLocally(exp => ({ ...exp, state: RunState.Active }));
          break;
        case 'archive':
          if (!isExperimentTask(task)) break;
          await archiveExperiment(parseInt(task.id), !task.archived);
          updateExperimentLocally(exp => ({ ...exp, archived: true }));
          break;
        case 'cancel':
          await setExperimentState({
            experimentId: parseInt(task.id),
            state: RunState.StoppingCanceled,
          });
          updateExperimentLocally(exp => ({ ...exp, state: RunState.StoppingCanceled }));
          break;
        case 'createTensorboard': {
          const tensorboard = await createTensorboard({
            ids: [ parseInt(task.id) ],
            type: TBSourceType.Experiment,
          });
          openCommand(tensorboard);
          break;
        }
        case 'kill':
          await killTask(task);
          if (isExperiment) {
            // We don't provide immediate updates for command types yet.
            updateExperimentLocally(exp => ({ ...exp, state: RunState.StoppingCanceled }));
          }
          break;
        case 'pause':
          await setExperimentState({
            experimentId: parseInt(task.id),
            state: RunState.Paused,
          });
          updateExperimentLocally(exp => ({ ...exp, state: RunState.Paused }));
          break;
        case 'viewLogs': {
          const taskType = (task as CommandTask).type.toLocaleLowerCase();
          const path = `/det/${taskType}/${task.id}/logs?id=${task.name}`;
          openBlank(setupUrlForDev(path));
          break;
        }
      }
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: `Unable to ${params.key} task ${task.id}.`,
        publicSubject: `${capitalize(params.key)} failed.`,
        silent: false,
        type: ErrorType.Server,
      });
    }
    // TODO show loading indicator when we have a button component that supports it.
  };

  const menuItems: React.ReactNode[] = [];
  if (isResumable) menuItems.push(<Menu.Item key="activate">Activate</Menu.Item>);
  if (isPausable) menuItems.push(<Menu.Item key="pause">Pause</Menu.Item>);
  if (isArchivable) menuItems.push(<Menu.Item key="archive">Archive</Menu.Item>);
  if (isCancelable) menuItems.push(<Menu.Item key="cancel">Cancel</Menu.Item>);
  if (isKillable) menuItems.push(<Menu.Item key="kill">Kill</Menu.Item>);
  if (isExperiment) {
    menuItems.push(<Menu.Item key="createTensorboard">Open Tensorboard</Menu.Item>);
  } else {
    menuItems.push(<Menu.Item key="viewLogs">View Logs</Menu.Item>);
  }

  if (menuItems.length === 0) {
    return (
      <div className={css.base} title="No actions available" onClick={stopPropagation}>
        <button disabled>
          <Icon name="overflow-vertical" />
        </button>
      </div>
    );
  }

  const menu = <Menu onClick={handleMenuClick}>{menuItems}</Menu>;

  return (
    <div className={css.base} title="Open actions menu" onClick={stopPropagation}>
      <Dropdown overlay={menu} placement="bottomRight" trigger={[ 'click' ]}>
        <button onClick={stopPropagation}>
          <Icon name="overflow-vertical" />
        </button>
      </Dropdown>
    </div>
  );
};

export default TaskActionDropdown;
