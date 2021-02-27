import { isNumber } from 'util';

import { Dropdown, Menu } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React from 'react';

import Icon from 'components/Icon';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { paths } from 'routes/utils';
import {
  activateExperiment, archiveExperiment, cancelExperiment, killExperiment, killTask,
  openOrCreateTensorboard, pauseExperiment, unarchiveExperiment,
} from 'services/api';
import { AnyTask, CommandTask, ExperimentTask, RunState } from 'types';
import { capitalize } from 'utils/string';
import { isExperimentTask } from 'utils/task';
import { cancellableRunStates, isTaskKillable, terminalRunStates } from 'utils/types';
import { openCommand } from 'wait';

import Link from './Link';
import css from './TaskActionDropdown.module.scss';

export enum Action {
  Activate = 'activate',
  Archive = 'archive',
  Cancel = 'cancel',
  Kill = 'kill',
  Pause = 'pause',
  Tensorboard = 'tensorboard',
  Unarchive = 'unarchive',
}

interface Props {
  onComplete?: (action?: Action) => void;
  task: AnyTask;
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
      const action = params.key as Action;
      switch (action) { // Cases should match menu items.
        case Action.Activate:
          await activateExperiment({ experimentId: id });
          if (onComplete) onComplete(action);
          break;
        case Action.Archive:
          if (!isExperiment) break;
          await archiveExperiment({ experimentId: id });
          if (onComplete) onComplete(action);
          break;
        case Action.Cancel:
          await cancelExperiment({ experimentId: id });
          if (onComplete) onComplete(action);
          break;
        case Action.Tensorboard: {
          const tensorboard = await openOrCreateTensorboard({ experimentIds: [ id ] });
          openCommand(tensorboard);
          break;
        }
        case Action.Kill:
          if (isExperiment) {
            await killExperiment({ experimentId: id });
            if (onComplete) onComplete(action);
          } else {
            await killTask(task as CommandTask);
          }
          break;
        case Action.Pause:
          await pauseExperiment({ experimentId: id });
          if (onComplete) onComplete(action);
          break;
        case Action.Unarchive:
          if (!isExperiment) break;
          await unarchiveExperiment({ experimentId: id });
          if (onComplete) onComplete(action);
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
  if (isResumable) menuItems.push(<Menu.Item key={Action.Activate}>Activate</Menu.Item>);
  if (isPausable) menuItems.push(<Menu.Item key={Action.Pause}>Pause</Menu.Item>);
  if (isArchivable) menuItems.push(<Menu.Item key={Action.Archive}>Archive</Menu.Item>);
  if (isUnarchivable) menuItems.push(<Menu.Item key={Action.Unarchive}>Unarchive</Menu.Item>);
  if (isCancelable) menuItems.push(<Menu.Item key={Action.Cancel}>Cancel</Menu.Item>);
  if (isKillable) menuItems.push(<Menu.Item key={Action.Kill}>Kill</Menu.Item>);
  if (isExperiment) {
    menuItems.push(<Menu.Item key={Action.Tensorboard}>View in TensorBoard</Menu.Item>);
  } else {
    menuItems.push(<Menu.Item key="viewLogs">
      <Link path={paths.taskLogs(task as CommandTask)}>View Logs</Link>
    </Menu.Item>);
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
