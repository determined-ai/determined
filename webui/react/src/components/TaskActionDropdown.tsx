import { isNumber } from 'util';

import { Dropdown, Menu } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React from 'react';

import Icon from 'components/Icon';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { openCommand } from 'routes/utils';
import {
  activateExperiment, archiveExperiment, cancelExperiment, killExperiment, killTask,
  openOrCreateTensorboard, pauseExperiment, unarchiveExperiment,
} from 'services/api';
import { AnyTask, CommandTask, ExperimentTask, RunState, TBSourceType } from 'types';
import { capitalize } from 'utils/string';
import { isExperimentTask } from 'utils/task';
import { cancellableRunStates, isTaskKillable, terminalRunStates } from 'utils/types';

import Link from './Link';
import css from './TaskActionDropdown.module.scss';

interface Props {
  task: AnyTask;
  onComplete?: () => void;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const taskPath = (task: CommandTask): string => {
  const taskType = task.type.toLocaleLowerCase();
  return`/${taskType}/${task.id}/logs?id=${task.name}`;
};

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
          await activateExperiment({ experimentId: id });
          if (onComplete) onComplete();
          break;
        case 'archive':
          if (!isExperiment) break;
          await archiveExperiment({ experimentId: id });
          if (onComplete) onComplete();
          break;
        case 'cancel':
          await cancelExperiment({ experimentId: id });
          if (onComplete) onComplete();
          break;
        case 'openOrCreateTensorboard': {
          const tensorboard = await openOrCreateTensorboard({
            ids: [ id ],
            type: TBSourceType.Experiment,
          });
          openCommand(tensorboard);
          break;
        }
        case 'kill':
          if (isExperiment) {
            await killExperiment({ experimentId: id });
            if (onComplete) onComplete();
          } else {
            await killTask(task as CommandTask);
          }
          break;
        case 'pause':
          await pauseExperiment({ experimentId: id });
          if (onComplete) onComplete();
          break;
        case 'unarchive':
          if (!isExperiment) break;
          await unarchiveExperiment({ experimentId: id });
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
    menuItems.push(<Menu.Item key="openOrCreateTensorboard">View in TensorBoard</Menu.Item>);
  } else {
    menuItems.push(<Menu.Item key="viewLogs">
      <Link path={taskPath(task as CommandTask)}>View Logs</Link>
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
