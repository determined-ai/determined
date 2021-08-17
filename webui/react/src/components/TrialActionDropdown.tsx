import { Dropdown, Menu } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React from 'react';

import Icon from 'components/Icon';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { paths, routeToReactUrl } from 'routes/utils';
import { openOrCreateTensorboard } from 'services/api';
import { ExperimentAction as Action, TrialItem } from 'types';
import { capitalize } from 'utils/string';
import { openCommand } from 'wait';

import css from './TaskActionDropdown.module.scss';

interface Props {
  experimentId: number;
  trial: TrialItem;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const TrialActionDropdown: React.FC<Props> = ({ trial, experimentId }: Props) => {
  const handleMenuClick = async (params: MenuInfo): Promise<void> => {
    params.domEvent.stopPropagation();
    try {
      const action = params.key as Action;
      switch (action) { // Cases should match menu items.
        case Action.ViewLogs:
          routeToReactUrl(paths.trialLogs(trial.id, experimentId));
          break;
        case Action.OpenTensorBoard:
          openCommand(await openOrCreateTensorboard({ trialIds: [ trial.id ] }));
          break;
      }
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: `Unable to ${params.key} trial ${trial.id}.`,
        publicSubject: `${capitalize(params.key.toString())} failed.`,
        silent: false,
        type: ErrorType.Server,
      });
    }
  };
  const menuItems: React.ReactNode[] = [];
  menuItems.push(<Menu.Item key={Action.OpenTensorBoard}>View in TensorBoard</Menu.Item>);
  menuItems.push(<Menu.Item key={Action.ViewLogs}>View Logs</Menu.Item>);

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
