import { isNumber } from 'util';

import { Dropdown, Menu } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React from 'react';

import Icon from 'components/Icon';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { archiveExperiment, createTensorboard, killTask, setExperimentState } from 'services/api';
import { AnyTask, CommandTask, ExperimentTask, RunState, TBSourceType } from 'types';
import { openBlank, openCommand } from 'utils/routes';
import { capitalize } from 'utils/string';
import { isExperimentTask } from 'utils/task';
import { cancellableRunStates, isTaskKillable, terminalRunStates } from 'utils/types';

import css from './TaskActionDropdown.module.scss';

interface Props {
  task: AnyTask;
  onComplete?: () => void;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const TaskActionDropdown: React.FC<Props> = ({ task, onComplete }: Props) => {
  const id = isNumber(task.id) ? task.id : parseInt(task.id);
  const isExperiment = isExperimentTask(task);
  const isExperimentTerminal = terminalRunStates.has(task.state as RunState);
  const isArchivable = isExperiment && isExperimentTerminal && !(task as ExperimentTask).archived;
  const isUnarchivable = isExperiment && isExperimentTerminal && (task as ExperimentTask).archived;
  const isKillable = isTaskKillable(task);
  const isPausable = isExperiment
    && task.state === RunState.Active;
  const isResumable = isExperiment
    && task.state === RunState.Paused;
  const isCancelable = isExperiment
    && cancellableRunStates.includes(task.state as RunState);

  const handleMenuClick = async (params: MenuInfo): Promise<void> => {
    params.domEvent.stopPropagation();
    try {
      switch (params.key) { // Cases should match menu items.
        case 'activate':
          await setExperimentState({
            experimentId: id,
            state: RunState.Active,
          });
          if (onComplete) onComplete();
          break;
        case 'archive':
          if (!isExperimentTask(task)) break;
          await archiveExperiment(id);
          if (onComplete) onComplete();
          break;
        case 'cancel':
          await setExperimentState({
            experimentId: id,
            state: RunState.StoppingCanceled,
          });
          if (onComplete) onComplete();
          break;
        case 'createTensorboard': {
          const tensorboard = await createTensorboard({
            ids: [ id ],
            type: TBSourceType.Experiment,
          });
          openCommand(tensorboard);
          break;
        }
        case 'kill':
          await killTask(task);
          if (isExperiment && onComplete) onComplete();
          break;
        case 'pause':
          await setExperimentState({
            experimentId: id,
            state: RunState.Paused,
          });
          if (onComplete) onComplete();
          break;
        case 'viewLogs': {
          const taskType = (task as CommandTask).type.toLocaleLowerCase();
          const path = `/det/${taskType}/${task.id}/logs?id=${task.name}`;
          openBlank(path);
          break;
        }
        case 'unarchive':
          if (!isExperimentTask(task)) break;
          await archiveExperiment(id, false);
          if (onComplete) onComplete();
      }
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: `Unable to ${params.key} task ${task.id}.`,
        publicSubject: `${capitalize(params.key.toString())} failed.`,
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
  if (isUnarchivable) menuItems.push(<Menu.Item key="unarchive">Unarchive</Menu.Item>);
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
