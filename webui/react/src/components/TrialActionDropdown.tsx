import { isNumber } from 'util';

import { Dropdown, Menu } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React from 'react';

import Icon from 'components/Icon';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { paths, routeToReactUrl } from 'routes/utils';
import {
  activateExperiment, archiveExperiment, cancelExperiment, killExperiment, killTask,
  openOrCreateTensorboard, pauseExperiment, unarchiveExperiment,
} from 'services/api';
import { AnyTask, CommandTask, ExperimentBase, ExperimentTask, RunState, TrialItem } from 'types';
import { capitalize } from 'utils/string';
import { isExperimentTask } from 'utils/task';
import { cancellableRunStates, isTaskKillable, terminalRunStates } from 'utils/types';
import { openCommand } from 'wait';

import Link from './Link';
import css from './TaskActionDropdown.module.scss';

export enum Action {
  Tensorboard = 'tensorboard',
  Logs = 'logs'
}

interface Props {
  experiment: ExperimentBase;
  onComplete?: (action?: Action) => void;
  trial: TrialItem;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const TrialActionDropdown: React.FC<Props> = ({ trial, experiment, onComplete }: Props) => {
  const handleMenuClick = async (params: MenuInfo): Promise<void> => {
    params.domEvent.stopPropagation();
    try {
      const action = params.key as Action;
      switch (action) { // Cases should match menu items.
        case Action.Logs:
          routeToReactUrl(paths.trialLogs(trial.id, experiment.id));
          if (onComplete) onComplete(action);
          break;
        case Action.Tensorboard:
          openCommand(await openOrCreateTensorboard({ trialIds: [ trial.id ] }));
          if (onComplete) onComplete(action);
          break;
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
  menuItems.push(<Menu.Item key={Action.Tensorboard}>Tensorboard</Menu.Item>);
  menuItems.push(<Menu.Item key={Action.Logs}>View Logs</Menu.Item>);

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

export default TrialActionDropdown;
